# coding: utf8
import unittest
from groups import OpGroup, setup_pprof_ops
from groups import check_log_args, check_metric_args
from operation import Op
from collectors import Collector


class TestOpGroup(unittest.TestCase):
    def test_iter(self):
        # an empty group
        for c in OpGroup('basic'):
            self.assertEqual(c, None)
        # a group of base collectors
        for c in OpGroup('basic', [Collector(),
                                   Collector(), Collector()]):
            self.assertEqual(c.collect(), NotImplemented)

    def test_init(self):
        group = OpGroup('basic')
        self.assertEqual(group.name, 'basic')
        self.assertEqual(group.ops, [])

        ops = [Op(None, None)]
        group = OpGroup('test', ops)
        self.assertEqual(group.name, 'test')
        self.assertEqual(group.ops, ops)

    def test_add_ops(self):
        group = OpGroup('test')
        self.assertEqual(group.ops, [])
        ops = [Op(None, None)]
        group.add_ops(ops)
        self.assertEqual(group.ops, ops)

    def test_get_ops(self):
        group = OpGroup('test')
        self.assertEqual(group.ops, [])
        self.assertEqual(group.get_ops(), [])

        ops = [Op(None, None)]
        group = OpGroup('test', ops)
        self.assertEqual(group.ops, ops)
        self.assertEqual(group.get_ops(), ops)


class Object:
    pass


def setup_args(*attrs):
    args = Object()
    for attr in attrs:
        setattr(args, attr, None)
    return args


def setup_arg_values(**attrs):
    args = Object()
    for attr, val in attrs.items():
        setattr(args, attr, val)
    return args


class TestUtil(unittest.TestCase):
    def test_check_log_args(self):
        args = setup_args('log_dir', 'log_spliter', 'begin', 'end')
        with self.assertRaises(SystemExit):
            check_log_args(args)

        args.log_dir = '/tmp/test'
        with self.assertRaises(SystemExit):
            check_log_args(args)

        args.log_spliter = '/usr/local/bin/spliter'
        with self.assertRaises(SystemExit):
            check_log_args(args)

        args.begin = '2019'
        with self.assertRaises(SystemExit):
            check_log_args(args)

        # no exception raised
        args.end = '2020'
        check_log_args(args)

    def test_check_metric_args(self):
        args = setup_args('begin', 'end')
        with self.assertRaises(SystemExit):
            check_metric_args(args)

        args.begin = '2019'
        with self.assertRaises(SystemExit):
            check_metric_args(args)

        # no exception raised
        args.end = '2020'
        check_metric_args(args)

    def test_collect_args(self):
        from groups import collect_args

        args = setup_args('name', 'age', 'home', 'love')
        args.name = 'Raymond'
        args.age = 1024
        args.home = 'Beijing'
        args.love = None

        v = collect_args(args)
        self.assertEqual(
            v, {'name': 'Raymond', 'age': 1024, 'home': 'Beijing'})

    def test_setup_op_groups(self):
        from groups import setup_op_groups
        topology = {
            'cluster_name': 'unit test',
            'status': 'success',
            'hosts': [{
                'ip': '127.0.0.1',
                'status': 'success',
                'user': 'tidb',
                'components': [{
                    'name': 'tidb',
                    'status': 'success',
                    'deploy_dir': '/home/tidb/deploy',
                    'status_port': '10080',
                }, {
                    'name': 'tikv',
                    'status': 'success',
                    'deploy_dir': '/home/tidb/deploy',
                    'port': '20016',
                }, {
                    'name': 'pd',
                    'status': 'success',
                    'deploy_dir': '/home/tidb/deploy',
                    'port': 2379,
                }, {
                    'name': 'prometheus',
                    'status': 'success',
                    'deploy_dir': '/home/tidb/deploy',
                    'port': '9090'
                }]
            }],
        }

        args = setup_arg_values(inspection_id='1',
                                data_dir='/tmp/test',
                                collect='basic:profile:dbinfo:config:metric:log',
                                log_dir='/tmp/test/log',
                                log_spliter='/tmp/bin/spliter',
                                begin='2019',
                                end='2020',
                                )
        groups = setup_op_groups(topology, args)

        self.assertEqual(len(groups), 8)
        self.assertEqual(len(groups['_setup'].get_ops()), 3)
        self.assertEqual(len(groups['_teardown'].get_ops()), 1)

        # 6 os ops, 1 insight
        self.assertEqual(len(groups['basic'].get_ops()), 7)

        # pprof: 6 pd, 6 tidb; perf: 1 tikv
        self.assertEqual(len(groups['profile'].get_ops()), 13)

        # 1 for tidb
        self.assertEqual(len(groups['dbinfo'].get_ops()), 1)

        # 1 pd, 1 tikv, 1 tidb
        self.assertEqual(len(groups['config'].get_ops()), 3)

        # 1 metric op which will raise a exception, 1 alert
        self.assertEqual(len(groups['metric'].get_ops()), 2)

        # 1 log
        self.assertEqual(len(groups['log'].get_ops()), 1)

    def test_setup_pprof_ops(self):
        ops = setup_pprof_ops('127.0.0.1:6060')
        self.assertEqual(len(ops), 6)

        names = (op.collector.name for op in ops)
        for name in names:
            print name
            self.assertNotEqual(name, "")
            self.assertRegexpMatches(name, ".*profile$")
        addrs = (op.collector.addr for op in ops)
        for addr in addrs:
            print addr
            self.assertEqual(addr, '127.0.0.1:6060')


if __name__ == '__main__':
    unittest.main()
