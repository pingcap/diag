# coding:utf8

from collectors.output import FileOutput, NullOutput


class Op:
    def __init__(self, collector, output, f=None):
        self.collector = collector
        self.output = output
        self.f = f

    def do(self):
        # f will be executed before collecting, it can do some init work
        if self.f:
            self.f()
        c = self.collector
        out = self.output
        out.output(c.collect())
