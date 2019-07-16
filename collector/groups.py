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
            if status != 'success':
                logging.warn('skip host:%s service:%s status:%s',
                             ip, svc, status)
                continue

            name = svc['name']
            if name == 'tidb':
                status_port = svc['status_port']
                addr = "%s:%s" % (ip, status_port)
                groups['pprof'].add_ops(
                    setup_pprof_ops(inspection_id, addr, datadir+'/pprof'))
            if name == 'tikv':
                pass
            if name == 'pd':
                pass
            if name == 'prometheus':
                pass
            if name == 'alertmanager':
                pass
    return groups


def setup_pprof_ops(inspection_id, addr='127.0.0.1:6060', basedir='pprof'):
    """Setup all pprof related collectors for a host"""
    join = os.path.join

    def op(cls, filename):
        return Op(cls(addr=addr), FileOutput(join(basedir, filename)))

    ops = [
        op(CPUProfileCollector, '%s-%s-cpu.pb.gz' % (addr, inspection_id)),
        op(MemProfileCollector, '%s-%s-mem.pb.gz' % (addr, inspection_id)),
        op(BlockProfileCollector, '%s-%s-block.pb.gz' % (addr, inspection_id)),
        op(AllocsProfileCollector, '%s-%s-allocs.pb.gz' % (addr, inspection_id)),
        op(MutexProfileCollector, '%s-%s-mutex.pb.gz' % (addr, inspection_id)),
        op(ThreadCreateProfileCollector, '%s-%s-threadcreate.pb.gz' %
           (addr, inspection_id)),
        op(TraceProfileCollector, '%s-%s-trace.pb.gz' % (addr, inspection_id))
    ]
    return ops
