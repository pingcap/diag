# coding:utf8
import unittest
import json

from codec import Codec, RawDecoder, RawEncoder
from codec import JSONEncoder, JSONDecoder


class TestCodec(unittest.TestCase):
    def test_raw_raw(self):
        c = Codec(RawEncoder(), RawDecoder())
        self.assertEqual(c.encode('a'), 'a')
        self.assertEqual(c.decode('a'), 'a')

    def test_raw_json(self):
        s = '{"name": "tom", "age": 18}'
        obj = {"name": "tom", "age": 18}

        c = Codec(RawEncoder(), JSONDecoder())
        self.assertEqual(c.encode(s), s)
        self.assertEqual(c.decode(s), obj)

    def test_json_raw(self):
        s = '{"name": "tom", "age": 18}'
        obj = {"name": "tom", "age": 18}

        c = Codec(JSONEncoder(), RawDecoder())
        self.assertEqual(c.encode(obj), json.dumps(obj))
        self.assertEqual(c.decode(s), s)

    def test_json_json(self):
        s = '{"name": "tom", "age": 18}'
        obj = {"name": "tom", "age": 18}

        c = Codec(JSONEncoder(), JSONDecoder())
        self.assertEqual(c.encode(obj), json.dumps(obj))
        self.assertEqual(c.decode(s), obj)
