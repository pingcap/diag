# coding: utf8
import unittest
from groups import OpGroup, setup_pprof_ops, parse_duration
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


class TestUtil(unittest.TestCase):
    def test_setup_pprof_ops(self):
        ops = setup_pprof_ops('127.0.0.1:6060')
        self.assertEqual(len(ops), 7)

        names = (op.collector.name for op in ops)
        for name in names:
            print name
            self.assertNotEqual(name, "")
            self.assertRegexpMatches(name, ".*profile$")
        addrs = (op.collector.addr for op in ops)
        for addr in addrs:
            print addr
            self.assertEqual(addr, '127.0.0.1:6060')

    def test_parse_duration(self):
        self.assertEqual(parse_duration('1h'), 3600)
        self.assertEqual(parse_duration('1m'), 60)
        self.assertEqual(parse_duration('1s'), 1)
        self.assertEqual(parse_duration('1'), 1)
        self.assertEqual(parse_duration('1h1m'), 3660)
        self.assertEqual(parse_duration('1h1m1s'), 3661)
        self.assertEqual(parse_duration('1h1m1'), 3661)
        self.assertEqual(parse_duration('3661'), 3661)
        self.assertEqual(parse_duration(3661), 3661)


if __name__ == '__main__':
    unittest.main()
