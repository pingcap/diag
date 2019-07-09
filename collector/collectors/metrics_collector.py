from collector import Collector


class MetricsCollector(Collector):
    def __init__(self, name='metrics'):
        self.name = name

    def collect(self):
        return 'metrics'


if __name__ == '__main__':
    print(MetricsCollector().collect())
