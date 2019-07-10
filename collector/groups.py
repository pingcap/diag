# coding: utf8
from collectors import *


class CollectorGroup:
    def __init__(self, name, collectors={}):
        self.name = name
        self.collectors = collectors

    def get_collectors(self):
        return self.collectors

    def __iter__(self):
        return iter(self.collectors)


groups = {
    'basic': CollectorGroup('basic',
                            [Collector()]),
    'metrics': CollectorGroup('metrics',
                              [MetricsCollector()])
}
