#!/usr/bin/env python

import sys
import os


def validate_int(port):
    """
    port: expect to be the port argument like "9527", panic if port is not a \
    number larger than 0.
    return: `int` of the port
    """
    try:
        return int(port)
    except ValueError as e:
        print('Port {} is not available'.format(port))
        raise e


def service_name_stringer(service_name, port_id):
    """
    service_name: a `str` for service name, like "prometheus" or "influxd".
    port_id: an `int` for port id.
    return: an `str` represents the service name.
    """
    return "{}-{}".format(service_name, port_id)


def start_service(service_name, port_id):
    """
    Run `systemctl start` for the scripts.
    service_name: a `str` for service name, like "prometheus" or "influxd".
    port_id: an `int` for port id.
    """
    assert type(port_id) == int
    assert type(service_name) == str
    script_str = "systemctl start {}".format(
        service_name_stringer(service_name, port_id))
    print(script_str)
    os.system(script_str)


if __name__ == '__main__':
    (foresight_port, influxd_port,
     prometheus_port) = map(validate_int, sys.argv[1:])

    start_service('prometheus', prometheus_port)
    start_service('influxd', influxd_port)
    start_service('foresight', foresight_port)

    # message for starting the system
    print("""To start tidb-foresight (will listen on port {foresight_port}):
        systemctl start foresight-{foresight_port}
        systemctl start influxd-{influxd_port}
        systemctl start prometheus-{prometheus_port}\n""".format(
        foresight_port=foresight_port,
        influxd_port=influxd_port,
        prometheus_port=prometheus_port))
    print("""View the log as follows:
        journalctl -u foresight-{foresight_port}
        journalctl -u influxd-{influxd_port}
        journalctl -u prometheus-{prometheus_port}""".format(
        foresight_port=foresight_port,
        influxd_port=influxd_port,
        prometheus_port=prometheus_port))
