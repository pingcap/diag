#!/usr/bin/env python
# -*- coding=utf-8 -*-
"""
This script try to list the service file, and try to stop all the \
services in the system. The system will try to list all the files under \
the current directory.
"""

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


def stop_service(service_name, port_id):
    """
    Run `systemctl start` for the scripts.
    service_name: a `str` for service name, like "prometheus" or "influxd".
    port_id: an `int` for port id.
    """
    assert type(port_id) == int
    assert type(service_name) == str
    script_str = "systemctl stop {}".format(
        service_name_stringer(service_name, port_id))
    os.system(script_str)


if __name__ == "__main__":
    (foresight_port, influxd_port,
     prometheus_port) = map(validate_int, sys.argv[1:])

    stop_service('prometheus', prometheus_port)
    stop_service('influxd', influxd_port)
    stop_service('foresight', foresight_port)
