# coding: utf8
import logging
import os

from collectors.profile_collector import *
from collectors.output import *
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


def setup_ops(topology, datadir):
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
            if status != 'success':
                logging.warn('skip host:%s service:%s status:%s',
                             host, svc, status)
                continue

            name = svc['name']
            if name == 'tidb':
                status_port = svc['status_port']
                addr = "%s:%d" % (host, status_port)
                groups['pprof'].add_ops(setup_pprof_ops(addr, datadir))
            if name == 'tikv':
                pass
            if name == 'pd':
                pass
            if name == 'prometheus':
                pass
            if name == 'alertmanager':
                pass
    return groups


def setup_pprof_ops(addr='http://127.0.0.1:6060', basedir='pprof'):
    """Setup all pprof related collectors for a host"""
    join = os.path.join

    def op(cls, filename):
        return Op(cls(addr=addr), FileOutput(join(basedir, filename)))
    ops = [
        op(CPUProfileCollector, addr+'-cpu.pb.gz'),
        op(MemProfileCollector, addr+'-mem.pb.gz'),
        op(BlockProfileCollector, addr+'-block.pb.gz'),
        op(AllocsProfileCollector, addr+'-allocs.pb.gz'),
        op(MutexProfileCollector, addr+'-mutex.pb.gz'),
        op(ThreadCreateProfileCollector, addr+'-threadcreate.pb.gz'),
        op(TraceProfileCollector, addr+'-trace.pb.gz')
    ]
    return ops
