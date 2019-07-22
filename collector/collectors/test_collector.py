# coding: utf8
import unittest
from collector import Collector, HTTPCollector


class TestCollector(unittest.TestCase):
    def test_collect(self):
        self.assertEqual(Collector('cpu').collect(), NotImplemented)


class TestHTTPCollector(unittest.TestCase):
    def test_url(self):
        c = HTTPCollector(name='httpcollector', addr='127.0.0.1:8080',
                          path='/test', params={'k': 'v'})
        self.assertEqual(c._url(), "http://127.0.0.1:8080/test?k=v")

    def test_repr(self):
        c = HTTPCollector(name='httpcollector', addr='127.0.0.1:8080',
                          path='/test', params={'k': 'v'})
        self.assertNotEqual(repr(c), '')

    def test_collect(self):
        # TODO mock the HTTP response
        c = HTTPCollector(name='httpcollector', addr='127.0.0.1:8080',
                          path='/test', params={'k': 'v'})


if __name__ == '__main__':
    unittest.main()
