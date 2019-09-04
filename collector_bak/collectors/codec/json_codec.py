# coding: utf8
import json


class JSONEncoder:
    def encode(self, obj):
        return json.dumps(obj)


class JSONDecoder:
    def decode(self, data):
        return json.loads(data)
