# coding:utf8

from collectors.output import FileOutput, NullOutput


class Op:
    def __init__(self, collector, output):
        self.collector = collector
        self.output = output

    def do(self):
        c = self.collector
        out = self.output
        out.output(c.collect())
