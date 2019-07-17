# coding:utf8
import json

from collector import Collector


class VarCollector(Collector):
    def __init__(self, name='var', var_name=None, var_value=None):
        self.var_name = var_name
        self.var_value = var_value

    def collect(self):
        d = {self.var_name: self.var_value}
        json.dumps(d)
