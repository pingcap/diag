# coding: utf8
import logging
import os
import time

from collectors.profile_collector import *
from collectors.output import *
from collectors.metric_collector import *
from operation import Op


class OpGroup:
    def __init__(self, name, ops=[]):
        self.name = name
        self.ops = ops

    def get_ops(self):
        return self.ops

    def add_ops(self, ops=[]):
        self.ops += ops

    def __iter__(self):
        return iter(self.ops)


def setup_op_groups(topology, datadir, inspection_id, target):
    items = map(lambda x: x.split(':'), target.split(','))
    groups = {
        'basic': OpGroup('basic'),
        'pprof': OpGroup('pprof'),
        'hardware': OpGroup('hardware'),
        'metric': OpGroup('metric'),
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

    for host in hosts:
        status = host['status']
        ip = host['ip']
        user = host['user']
        services = host['components']
        logging.debug("host:%s status:%s user:%s", ip, status, user)
        for svc in services:
            status = svc['status']
            name = svc['name']
            if status != 'success':
                logging.warn('skip host:%s service:%s status:%s',
                             ip, name, status)
                continue

            if name == 'tidb':
                status_port = svc['status_port']
                addr = "%s:%s" % (ip, status_port)
                basedir = os.path.join(
                    datadir, inspection_id, 'pprof', addr, 'tidb')
                groups['pprof'].add_ops(
                    setup_pprof_ops(addr, basedir))
            if name == 'tikv':
                pass
            if name == 'pd':
                pass
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


def setup_alert_ops(addr='127.0.0.1:9090', basedir='alert'):
    filename = os.path.join(basedir, 'alert.json')
    return [Op(AlertCollector(addr=addr), FileOutput(filename))]


def parse_duration(duration):
    seconds = 0
    for c in str(duration):
        if '0' <= c <= '9':
            seconds = seconds * 10 + (ord('c')-ord('0'))
        elif c in ('h', 'H'):
            seconds = seconds * 3600
        elif c in ('m', 'M'):
            seconds = seconds * 60
        elif c == 's':
            pass
        else:
            raise Exception("invalid format")
    return seconds
