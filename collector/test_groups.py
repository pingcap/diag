# coding: utf8
import unittest
from groups import CollectorGroup
from collector import Collector


class TestCollectorGroup(unittest.TestCase):
    def test_iter(self):
        # an empty group
        for c in CollectorGroup('basic'):
            self.assertEqual(c, None)
        # a group of base collectors
        for c in CollectorGroup('basic', [Collector(),
                                          Collector(), Collector()]):
            self.assertEqual(c.collect(), NotImplemented)


if __name__ == '__main__':
    unittest.main()
