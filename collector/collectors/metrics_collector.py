# coding: utf8
from collector import Collector, HTTPCollector


class MetricsCollector(HTTPCollector):
    def __init__(self, name='metrics', addr='127.0.0.1:9000', start=None,
                 end=None):
        HTTPCollector.__init__(self, name, addr, '/api/v1')

    def collect(self):
        return 'metrics'


if __name__ == '__main__':
    print(MetricsCollector().collect())
