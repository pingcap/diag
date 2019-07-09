import unittest
from collector import Collector


class TestCollector(unittest.TestCase):
    def test_collect(self):
        self.assertEqual(Collector('cpu').collect(), NotImplemented)

if __name__ == '__main__':
    unittest.main()