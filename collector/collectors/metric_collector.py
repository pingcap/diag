# coding: utf8
import urlparse
import urllib2
import json
import time

from collector import Collector, HTTPCollector

min_step = 30


class MetricCollector(HTTPCollector):
    def __init__(self, name='metrics', addr='127.0.0.1:9090', metric='up',
                 path='/api/v1/', start=None, end=None):
        now = int(time.time())
        if start == None:
            start = now - 3600  # 1h
        if end == None:
            end = now
        # fixed to 60 points, so it is responsive to the time range
        step = (end-start)/60
        if step < min_step:
            step = min_step

        params = {
            'query': metric,
            'start': start,
            'end': end,
            'step': step
        }
        HTTPCollector.__init__(
            self, name, addr, path, params)


def get_metrics(addr):
    url = 'http://' + addr + '/api/v1/label/__name__/values'
    return json.loads(urllib2.urlopen(url).read())


if __name__ == '__main__':
    print(MetricCollector().collect())
