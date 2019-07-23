# coding: utf8
import logging
import os
import time

from collectors.profile_collector import *
from collectors.output import *
from collectors.metric_collector import *
from collectors.db_collector import *
from collectors.var_collector import VarCollector
from collectors.command_collector import CommandCollector
from operation import Op


class OpGroup:
    def __init__(self, name, ops=None):
        self.name = name
        self.ops = [] if ops == None else ops

    def get_ops(self):
        return self.ops

    def add_ops(self, ops=[]):
        self.ops += ops

    def __iter__(self):
        return iter(self.ops)


def setup_op_groups(topology, datadir, inspection_id, target):
    items = map(lambda x: x.split(':'), target.split(','))
    groups = {
        '_setup': OpGroup('_setup'),
        'basic': OpGroup('basic'),
        'profile': OpGroup('profile'),
        'metric': OpGroup('metric'),
        'config': OpGroup('config'),
        'dbinfo': OpGroup('dbinfo'),
        '_teardown': OpGroup('_teardown'),
    }

    # for some targets, they come along with an option
    # Ex. metric:1h, slowlog:1h
    options = {}
    for item in items:
        if groups.has_key(item[0]):
            if len(item) == 2:
                options[item[0]] = item[1]
        else:
            raise Exception("unsupported target: "+item[0])

    cluster = topology['cluster_name']
    status = topology['status']
    hosts = topology['hosts']
    logging.info("cluster:%s status:%s", cluster, status)

    groups['_setup'].add_ops([
        Op(VarCollector(var_name='collect', var_value=target),
           FileOutput(os.path.join(datadir, inspection_id, 'collect.json'))),
        Op(VarCollector(var_name='topology', var_value=topology),
           FileOutput(os.path.join(datadir, inspection_id, "topology.json")))])
    create = start = time.time()
    groups['_teardown'].add_ops(setup_meta_ops(
        cluster, os.path.join(datadir, inspection_id), create, start))

    db_collected = False
    deploydir = {}
    ips = [host['ip'] for host in hosts]
    for i in range(len(ips)):
        deploydir[ips[i]] = hosts[i]['components'][0]['deploy_dir']

    for host in hosts:
        status = host['status']
        ip = host['ip']
        user = host['user']
        services = host['components']
        logging.debug("host:%s status:%s user:%s", ip, status, user)

        groups['basic'].add_ops(setup_os_ops(ip,
                                             os.path.join(datadir, inspection_id)))
        groups['basic'].add_ops(setup_insight_ops(ip,
                                                  os.path.join(datadir,
                                                               inspection_id),
                                                  deploydir[ip]))
        for svc in services:
            status = svc['status']
            name = svc['name']
            deploydir = svc['deploy_dir']
            if status != 'success':
                logging.warn('skip host:%s service:%s status:%s',
                             ip, name, status)
                continue

            if name == 'tidb':
                status_port = svc['status_port']
                addr = "%s:%s" % (ip, status_port)

                # pprof collectors
                basedir = os.path.join(
                    datadir, inspection_id, 'profile', 'tidb', addr)
                groups['profile'].add_ops(
                    setup_pprof_ops(addr, basedir))

                # config collectors
                groups['config'].add_ops(
                    setup_conf_ops(ip,
                                   os.path.join(
                                       datadir, inspection_id, 'config'),
                                   deploydir, 'tidb'))

                # db collectors
                if not db_collected:
                    basedir = os.path.join(datadir, inspection_id, 'dbinfo')
                    groups['dbinfo'].add_ops(setup_db_ops(addr, basedir))
                    db_collected = True
            if name == 'tikv':
                addr = "%s:%s" % (ip, svc['port'])
                groups['config'].add_ops(
                    setup_conf_ops(ip,
                                   os.path.join(
                                       datadir, inspection_id, 'config'),
                                   deploydir, 'tikv'))
                groups['profile'].add_ops(
                    setup_perf_ops(ip,
                                   os.path.join(datadir, inspection_id,
                                                'profile', 'tikv', addr),
                                   deploydir))
            if name == 'pd':
                addr = "%s:%s" % (ip, svc['port'])
                groups['config'].add_ops(
                    setup_conf_ops(ip,
                                   os.path.join(
                                       datadir, inspection_id, 'config'),
                                   deploydir, 'pd'))
                basedir = os.path.join(
                    datadir, inspection_id, 'profile', 'pd', addr)
                groups['profile'].add_ops(
                    setup_pprof_ops(addr, basedir))
            if name == 'prometheus':
                port = svc['port']
                addr = "%s:%s" % (ip, port)
                basedir = os.path.join(datadir, inspection_id, 'metric')
                duration = options.setdefault('metric', '1h')
                groups['metric'].add_ops(
                    setup_metric_ops(addr, basedir, duration))
                groups['metric'].add_ops(setup_alert_ops(addr,
                                                         os.path.join(datadir, inspection_id)))
            if name == 'alertmanager':
                pass
    return groups


def setup_pprof_ops(addr='127.0.0.1:6060', basedir='pprof'):
    """Setup all pprof related collectors for a host"""
    join = os.path.join

    def op(cls, filename):
        return Op(cls(addr=addr), FileOutput(join(basedir, filename)))

    ops = [
        op(CPUProfileCollector, 'cpu.pb.gz'),
        op(MemProfileCollector, 'mem.pb.gz'),
        op(BlockProfileCollector, 'block.pb.gz'),
        op(AllocsProfileCollector, 'allocs.pb.gz'),
        op(MutexProfileCollector, 'mutex.pb.gz'),
        op(ThreadCreateProfileCollector, 'threadcreate.pb.gz'),
        op(TraceProfileCollector, 'trace.pb.gz')
    ]
    return ops


def setup_metric_ops(addr='127.0.0.1:9090', basedir='metric', duration='1h'):
    metrics = get_metrics(addr)
    if metrics['status'] != 'success':
        logging.error('get metrics failed, status:%s', metrics['status'])
        return

    ops = []
    join = os.path.join

    def op(metric):
        end = int(time.time())
        start = end - parse_duration(duration)

        # fixed to get 60 points when the time range is large
        step = (end - start)/60

        # the value should not be too small
        if step < 15:
            step = 15

        filename = join(basedir, "%s_%s_to_%s_%ss.json" %
                        (metric, start, end, step))
        return Op(MetricCollector(name=metric, addr=addr, metric=metric,
                                  path='/api/v1/query_range', start=start, end=end, step=step), FileOutput(filename))

    for m in metrics['data']:
        # skip the alerts, it is collected by the alert collector
        if m == 'ALERTS':
            continue
        ops.append(op(m))
    return ops


def setup_alert_ops(addr='127.0.0.1:9090', basedir='alert'):
    filename = os.path.join(basedir, 'alert.json')
    return [Op(AlertCollector(addr=addr), FileOutput(filename))]


def parse_duration(duration):
    seconds = 0
    part = 0
    for c in str(duration):
        if '0' <= c <= '9':
            part = part * 10 + (ord(c)-ord('0'))
        elif c in ('h', 'H'):
            seconds += part * 3600
            part = 0
        elif c in ('m', 'M'):
            seconds += part * 60
            part = 0
        elif c == 's':
            seconds += part
            part = 0
        else:
            raise Exception("invalid format")
    if part != 0:
        seconds += part
    return seconds


def setup_db_ops(addr='127.0.0.1:10080', basedir='dbinfo'):
    ops = []
    dbs = get_databases(addr)
    join = os.path.join

    def op(name):
        return Op(DBCollector(addr=addr, db=name), FileOutput(join(basedir,
                                                                   db+'.json')))
    for db in dbs:
        ops.append(op(db))
    return ops


def setup_os_ops(addr='127.0.0.1', basedir=''):
    join = os.path.join
    ops = [
        Op(CommandCollector(addr=addr, command='dmesg'),
           FileOutput(join(basedir, 'dmesg', addr, 'dmesg'))),
        Op(CommandCollector(addr=addr, command='netstat -s'),
           FileOutput(join(basedir, 'net', addr, 'netstat'))),
        Op(CommandCollector(addr=addr, command='iostat 1 60'),
           FileOutput(join(basedir, 'proc', addr, 'iostat_1_60'))),
        Op(CommandCollector(addr=addr, command='mpstat -P ALL 1 60'),
           FileOutput(join(basedir, 'proc', addr, 'mpstat_1_60'))),
        Op(CommandCollector(addr=addr, command='vmstat 1 60'),
           FileOutput(join(basedir, 'proc', addr, 'vmstat_1_60'))),
        Op(CommandCollector(addr=addr, command='pidstat -u -p ALL 1 60'),
           FileOutput(join(basedir, 'proc', addr, 'pidstat_1_60'))),
    ]
    return ops


def setup_insight_ops(addr='127.0.0.1', basedir='insight',
                      deploydir='/home/tidb/deploy'):
    join = os.path.join
    ops = [
        Op(CommandCollector(addr=addr, command=join(deploydir,
                                                    'scripts/tidb-insight/bin/collector')),
            FileOutput(join(basedir, 'insight', addr, 'collector.json'))),
    ]
    return ops


def setup_conf_ops(addr='127.0.0.1', basedir='conf',
                   deploydir='/home/tidb/deploy', service='tidb'):
    join = os.path.join
    filename = join(deploydir, 'conf', service+'.toml')
    cat = 'cat %s' % filename
    ops = [
        Op(CommandCollector(addr=addr, command=cat),
           FileOutput(join(basedir, addr, os.path.basename(filename)))),
    ]
    return ops


def setup_perf_ops(addr='127.0.0.1', basedir='profile',
                   deploydir='/home/tidb/deploy'):
    join = os.path.join
    # only support tikv now
    pidfile = join(deploydir, 'status/tikv.pid')
    perf = 'sudo perf record -F 99 -p `cat %s` -g -o /dev/stdout sleep 60' % (
        pidfile)
    ops = [
        Op(CommandCollector(addr=addr, command=perf),
           FileOutput(join(basedir, 'perf.data')))
    ]
    return ops


def setup_meta_ops(cluster_name, basedir, create, start):
    meta = {
        'cluster_name': cluster_name,
        'create_time': create,
        'inpect_time': start,
        'end_time': time.time()
    }
    join = os.path.join
    ops = [
        Op(VarCollector(var_name='meta', var_value=meta),
           FileOutput(join(basedir, 'meta.json')))
    ]
    return ops
