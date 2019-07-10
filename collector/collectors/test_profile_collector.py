import unittest

from profile_collector import PProfHTTPCollector, setup_pprof_collectors
from profile_collector import CPUProfileCollector, MemProfileCollector
from profile_collector import BlockProfileCollector, AllocsProfileCollector
from profile_collector import ThreadCreateProfileCollector, TraceProfileCollector
from profile_collector import MutexProfileCollector


class TestPProfHTTPCollector(unittest.TestCase):
    def test_url(self):
        # case of default path
        c = PProfHTTPCollector(addr='127.0.0.1:6060')
        self.assertEqual(c._url(), 'http://127.0.0.1:6060/debug/pprof')

        # case of customized path
        c = PProfHTTPCollector(addr='127.0.0.1:6060', path='/go/debug/pprof')
        self.assertEqual(c._url(), 'http://127.0.0.1:6060/go/debug/pprof')

    def test_collector_constructor(self):
        c = CPUProfileCollector()
        self.assertEqual(c.name, "cpuprofile")
        self.assertEqual(c.addr, '127.0.0.1:6060')
        self.assertEqual(c.path, '/debug/pprof/profile')
        self.assertEqual(c.params['seconds'], 10)

        c = MemProfileCollector()
        self.assertEqual(c.name, "memprofile")
        self.assertEqual(c.addr, '127.0.0.1:6060')
        self.assertEqual(c.path, '/debug/pprof/heap')

        c = BlockProfileCollector()
        self.assertEqual(c.name, "blockprofile")
        self.assertEqual(c.addr, '127.0.0.1:6060')
        self.assertEqual(c.path, '/debug/pprof/block')

        c = AllocsProfileCollector()
        self.assertEqual(c.name, "allocsprofile")
        self.assertEqual(c.addr, '127.0.0.1:6060')
        self.assertEqual(c.path, '/debug/pprof/allocs')

        c = ThreadCreateProfileCollector()
        self.assertEqual(c.name, "threadprofile")
        self.assertEqual(c.addr, '127.0.0.1:6060')
        self.assertEqual(c.path, '/debug/pprof/threadcreate')

        c = TraceProfileCollector()
        self.assertEqual(c.name, "traceprofile")
        self.assertEqual(c.addr, '127.0.0.1:6060')
        self.assertEqual(c.path, '/debug/pprof/trace')

        c = MutexProfileCollector()
        self.assertEqual(c.name, "mutexprofile")
        self.assertEqual(c.addr, '127.0.0.1:6060')
        self.assertEqual(c.path, '/debug/pprof/mutex')


class TestUtil(unittest.TestCase):
    def test_setup_pprof_collectors(self):
        collectors = setup_pprof_collectors('127.0.0.1:6060')
        self.assertEqual(len(collectors), 7)

        names = (c.name for c in collectors)
        for name in names:
            print name
            self.assertNotEqual(name, "")
            self.assertRegexpMatches(name, ".*profile$")
        addrs = (c.addr for c in collectors)
        for addr in addrs:
            print addr
            self.assertEqual(addr, '127.0.0.1:6060')
