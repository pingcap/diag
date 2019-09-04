# coding: utf8
from collector import Collector, HTTPCollector
from output import FileOutput
import urllib
import socket


class PProfHTTPCollector(HTTPCollector):
    """PProfHTTPCollector is a collector to
    collect profiles from go pprof
    """

    def __init__(self, name='pprof', addr='127.0.0.1:6060',
                 path='/debug/pprof', params={}):
        HTTPCollector.__init__(self, name, addr, path, params)

    def collect(self):
        seconds = self.params.get('seconds')
        if seconds != None:
            timeout = float(seconds) + 10
            socket.setdefaulttimeout(timeout)
        f = urllib.urlopen(self._url())
        return f.read()


class CPUProfileCollector(PProfHTTPCollector):
    def __init__(self, name='cpuprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/profile', params={'seconds': 10}):
        PProfHTTPCollector.__init__(self, name, addr, path, params)


class MemProfileCollector(PProfHTTPCollector):
    def __init__(self, name='memprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/heap'):
        PProfHTTPCollector.__init__(self, name, addr, path)


class BlockProfileCollector(PProfHTTPCollector):
    def __init__(self, name='blockprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/block'):
        PProfHTTPCollector.__init__(self, name, addr, path)


class AllocsProfileCollector(PProfHTTPCollector):
    def __init__(self, name='allocsprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/allocs'):
        PProfHTTPCollector.__init__(self, name, addr, path)


class GoroutineProfileCollector(PProfHTTPCollector):
    def __init__(self, name='goroutineprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/goroutine'):
        PProfHTTPCollector.__init__(self, name, addr, path)


class MutexProfileCollector(PProfHTTPCollector):
    def __init__(self, name='mutexprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/mutex'):
        PProfHTTPCollector.__init__(self, name, addr, path)


class ThreadCreateProfileCollector(PProfHTTPCollector):
    def __init__(self, name='threadprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/threadcreate'):
        PProfHTTPCollector.__init__(self, name, addr, path)


class TraceProfileCollector(PProfHTTPCollector):
    def __init__(self, name='traceprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/trace', params={}):
        PProfHTTPCollector.__init__(self, name, addr, path, params)


if __name__ == "__main__":
    c = CPUProfileCollector(addr='127.0.0.1:8000')
    print c
    content = c.collect()
    FileOutput("pprof/cpuprofile.pb.gz").output(content)
    c = MemProfileCollector(addr='127.0.0.1:8000')
    print c
    content = c.collect()
    FileOutput("pprof/memprofile.pb.gz").output(content)
