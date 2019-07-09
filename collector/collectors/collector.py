class Collector:
    """The base collector which define the necessary attributes and methods"""

    def __init__(self, name=None):
        """Return the base collector"""
        self.name = name

    def collect(self):
        """Collect the information"""
        return NotImplemented