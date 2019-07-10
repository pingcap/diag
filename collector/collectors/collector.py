# coding:utf8
import uuid
import urllib
import urllib2


class Collector:
    """The base collector which define the necessary attributes and methods"""

    def __init__(self, name=None):
        """Return the base collector"""
        self.name = name

    def collect(self):
        """Collect the information"""
        return NotImplemented


class HTTPCollector(Collector):
    def __init__(self, name='http', addr='172.0.0.1:80',
                 path='/', params={}):
        self.id = uuid.uuid4()
        self.name = name
        self.addr = addr
        self.path = path
        self.params = params

    def __repr__(self):
        return "%s %s %s" % (self.name, self.id, self.url())

    def _url(self):
        params = ""
        if len(self.params) > 0:
            params = "?" + urllib.urlencode(self.params)
        url = "http://%s%s%s" % (self.addr, self.path, params)
        return url

    def collect(self):
        f = urllib2.urlopen(self._url())
        return f.read()
