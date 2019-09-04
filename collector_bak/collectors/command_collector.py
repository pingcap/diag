# coding:utf8
from collector import Collector
import subprocess


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
        # work aroud bug of python2.7
        # https://bugs.python.org/issue9400
        try:
            return subprocess.check_output(command, shell=True)
        except subprocess.CalledProcessError as cpe:
            raise Exception(str(cpe))


if __name__ == "__main__":
    c = CommandCollector(command='echo hello world')
    print(c.collect())
