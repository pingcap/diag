from collector import Collector
import urllib
import urllib2
import uuid
import socket


class HTTPCollector(Collector):
    def __init__(self, name='http', addr='172.0.0.1:80',
                 path='/', params={}):
        self.id = uuid.uuid4()
        self.name = name
        self.addr = addr
        self.path = path
        self.params = params

    def __repr__(self):
        return "%s %s %s" % (self.id, self.name, self.url())

    def url(self):
        params = ""
        if len(self.params) > 0:
            params = "?" + urllib.urlencode(self.params)
        url = "http://%s%s%s" % (self.addr, self.path, params)
        return url

    def collect(self):
        f = urllib2.urlopen(self.url())
        return f.read()


class PProfHTTPCollector(HTTPCollector):
    def __init__(self, name='pprof', addr='127.0.0.1:6060',
                 path='/debug/pprof', params={}):
        HTTPCollector.__init__(self, name, addr, path, params)

    def collect(self):
        seconds = self.params.get('seconds')
        if seconds != None:
            timeout = float(seconds) + 10
            socket.setdefaulttimeout(timeout)
        f = urllib2.urlopen(self.url())
        return f.read()


class CPUProfileCollector(PProfHTTPCollector):
    def __init__(self, name='cpuprofile', addr='172.0.0.1:6060',
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


class TheadCreateProfileCollector(PProfHTTPCollector):
    def __init__(self, name='threadcreateprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/threadcreate'):
        PProfHTTPCollector.__init__(self, name, addr, path)


class TraceProfileCollector(PProfHTTPCollector):
    def __init__(self, name='traceprofile', addr='127.0.0.1:6060',
                 path='/debug/pprof/trace', params={}):
        PProfHTTPCollector.__init__(self, name, addr, path, params)


if __name__ == "__main__":
    c = CPUProfileCollector(addr='127.0.0.1:8000')
    print c
    print c.collect()

    c = MemProfileCollector(addr='127.0.0.1:8000')
    print c
    print c.collect()
