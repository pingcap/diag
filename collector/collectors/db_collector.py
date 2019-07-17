# coding:utf8
import json

from collector import HTTPCollector


class SchemaCollector(HTTPCollector):
    def __init__(self, name='db', addr='127.0.0.1:10080', path='/schema'):
        HTTPCollector.__init__(self, name, addr, path)


class DBCollector(HTTPCollector):
    def __init__(self, name='db', addr='127.0.0.1:10080', path='/schema',
                 db='mysql'):
        path = "%s/%s" % (path, db)
        HTTPCollector.__init__(self, name, addr, path)


def get_databases(addr):
    c = SchemaCollector(addr=addr)
    schema = json.loads(c.collect())
    return (db['dbname']['L'] for db in schema)
