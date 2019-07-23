# coding:utf8
import os

from collector import Collector


class CommandCollector(Collector):
    def __init__(self, name='remote', addr='', command=''):
        self.name = name
        self.addr = addr
        self.command = command

    def collect(self):
        command = self.command
        if self.addr:
            command = "ssh tidb@%s 'bash -c \"%s\"'" % (
                self.addr, self.command)
        with os.popen(command) as f:
            return f.read()


if __name__ == "__main__":
    c = CommandCollector(command='echo hello world')
    print(c.collect())
