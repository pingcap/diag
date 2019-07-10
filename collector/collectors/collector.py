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
        """Collect the information, it should not be
        called directly, it returns NotImplmented if you do so.
        """
        return NotImplemented


class HTTPCollector(Collector):
    """ HTTPCollector supplies a common patten to collect from an HTTP
    server.
    """

    def __init__(self, name='http', addr='172.0.0.1:80',
                 path='/', params={}):
        """Return an HTTP collector"""
        self.id = uuid.uuid4()
        self.name = name
        self.addr = addr
        self.path = path
        self.params = params

    def __repr__(self):
        """Represent a collector in a pretty format"""
        return "%s %s %s" % (self.name, self.id, self._url())

    def _url(self):
        """Construct an URL with addr, path and params"""
        params = ""
        if len(self.params) > 0:
            params = "?" + urllib.urlencode(self.params)
        url = "http://%s%s%s" % (self.addr, self.path, params)
        return url

    def collect(self):
        """Collect information from HTTP server"""
        f = urllib2.urlopen(self._url())
        return f.read()
