# coding: utf8
import logging
import os

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


def setup_op_groups(topology, datadir, inspection_id):
    groups = {
        'basic': OpGroup('basic'),
        'pprof': OpGroup('pprof'),
        'hardware': OpGroup('hardware'),
        'metric': OpGroup('metric'),
    }
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
                        datadir, inspection_idh, 'pprof', addr, 'tidb')
                groups['pprof'].add_ops(
                    setup_pprof_ops(addr, basedir))
            if name == 'tikv':
                pass
            if name == 'pd':
                pass
            if name == 'prometheus':
                pass
            if name == 'alertmanager':
                pass
    return groups


def setup_pprof_ops(addr='127.0.0.1:6060', basedir='pprof'):
    """Setup all pprof related collectors for a host"""
    join=os.path.join
    def op(cls, filename):
        return Op(cls(addr=addr), FileOutput(join(basedir, filename)))

    ops=[
        op(CPUProfileCollector, 'cpu.pb.gz'),
        op(MemProfileCollector, 'mem.pb.gz'),
        op(BlockProfileCollector, 'block.pb.gz'),
        op(AllocsProfileCollector, 'allocs.pb.gz'),
        op(MutexProfileCollector, 'mutex.pb.gz'),
        op(ThreadCreateProfileCollector, 'threadcreate.pb.gz'),
        op(TraceProfileCollector, 'trace.pb.gz')
    ]
    return ops


def setup_metric_ops(addr='127.0.0.1:9090', basedir='metric'):
    metrics=get_metrics(addr)
    if metrics['status'] != 'success':
        logging.error('get metrics failed, status:%s', metrics['status'])
        return

    ops=[]
    join=os.path.join

    def op(metric):
        filename=join(basedir, "%s.json" % metric)
        return Op(MetricCollector(name=metric, addr=addr),
                  FileOutput(filename=filename))

    for m in metrics['data']:
        ops.append(op(m))
