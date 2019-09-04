# coding: utf8
import urlparse
import urllib2
import json
import time

from collector import Collector, HTTPCollector
from rfc3339 import parse_datetime

# the prometheus can give at most 11000 points for every series
# so, for precision you can set MAX_POINTS with a higher value
# but no more than 11000.
MAX_POINTS = 11000

class MetricCollector(HTTPCollector):
    def __init__(self, name='metrics', addr='127.0.0.1:9090', metric='up',
                 path='/api/v1/query', start=None, end=None):
        now = int(time.time())
        step = 15
        if start is not None and end is not None:
            delta = parse_datetime(end) - parse_datetime(start)
            step = (delta.days * 24 * 60 * 60 + delta.seconds) / MAX_POINTS + 1
            if step < 15:   # the most accurate prometheus can give (15s a point)
                step = 15
        if start == None:
            start = now - 3600          # 1h
        if end == None:
            end = now        
        params = {
            'query': metric,
            'start': start,
            'end': end,
            'step': step
        }
        HTTPCollector.__init__(
            self, name, addr, path, params)


class AlertCollector(HTTPCollector):
    def __init__(self, name='alerts', addr='127.0.0.1:9090'):
        HTTPCollector.__init__(
            self, name, addr, '/api/v1/query', {'query': 'ALERTS'})


def get_metrics(addr):
    url = 'http://' + addr + '/api/v1/label/__name__/values'
    return json.loads(urllib2.urlopen(url).read())


if __name__ == '__main__':
    c = MetricCollector(addr='172.16.4.4:30648')
    print(c.collect())

    c = AlertCollector(addr='172.16.4.4:30648')
    print(c.collect())
