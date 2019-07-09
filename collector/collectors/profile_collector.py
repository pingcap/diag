from collector import Collector
import urllib


class HTTPCollector(Collector):
    def collect(self, addr='127.0.0.1:8804', path='', opts={}):
        url = "http://%s%s?%s" % (addr, path, urllib.urlencode(opts))
        f = urllib.urlopen(url)
        return f.read()


class CPUProfileCollector(HTTPCollector):
    def __init__(self, name='profile'):
        self.name = name

    def collect(self, addr='127.0.0.1:8804', path='/debug/pprof/profile',
                opts={'seconds': 10}):


if __name__ == "__main__":
    c = CPUProfileCollector()
    print c.collect('127.0.0.1:8000')
