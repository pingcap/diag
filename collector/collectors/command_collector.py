# coding:utf8
from collector import Collector
import os
import sys
if os.name == 'posix' and sys.version_info[0] < 3:
    import subprocess32 as subprocess
else:
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
        return subprocess.check_output(command, shell=True)


if __name__ == "__main__":
    c = CommandCollector(command='echo hello world')
    print(c.collect())
