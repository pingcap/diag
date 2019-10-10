#!/usr/bin/env python
# -*- coding=utf-8 -*-
"""
This script will be used in `make install`, it will:
1. Generate directory `data` `log` `conf`, and prepare data\log\config \
   directory for `influxd` and `prometheus`.
2. Move bin\web-dist\log\config to $prefix/tidb-foresight.
"""

import sys
import os
from string import Template


def mkdir_if_nonexists(path):
    """
    Mkdir for path if `path` directory not exists.
    path: str
    """
    if not os.path.exists(path):
        os.mkdir(path)


def generate_service(prefix):
    """
    generate directory for config.
    """
    # mkdir_if_nonexists('conf')
    arguments_dict = {'prefix': prefix}
    service_list = [('prometheus', 9527), ('influxd', 9528),
                    ('foresight', 9529)]

    # create and copy files for them
    for service_name, number in service_list:
        with open(
                'templates/{}.template.service'.format(service_name),
                mode='r') as template_s:
            src = Template(template_s.read())
            # generate prefix for config
            with open('{}-{}.service'.format(service_name, number), 'w+') as fw:
                fw.write(src.safe_substitute(arguments_dict))


def generate_conf(prefix):
    mkdir_if_nonexists('conf')
    arguments_dict = {'prefix': prefix}
    service_list = ['influxdb.conf', 'prometheus.yml']

    # create and copy files for them
    for service_name in service_list:
        with open(
                'templates/conf/{}'.format(service_name),
                mode='r') as template_s:
            src = Template(template_s.read())
            # generate prefix for config
            with open('conf/{}'.format(service_name), 'w+') as fw:
                fw.write(src.safe_substitute(arguments_dict))


if __name__ == '__main__':
    prefix = sys.argv[1]
    # move all to prefix
    # if prefix is a subdirectory of current dir, it will fail
    print(os.path.realpath(prefix), os.path.realpath('.'))
    if os.path.realpath('.') in os.path.realpath(prefix):
        print('prefix is a subdirectory of src files, cannot work')
        exit(1)

    generate_service(prefix)
    generate_conf(prefix)

    directories = [
        'data', 'data/influxdb', 'data/prometheus', 'log', 'log/influxdb',
        'log/prometheus'
    ]
    for to_create_dir in directories:
        mkdir_if_nonexists(to_create_dir)

    # final stage for copying files
    # need to check and create if prefix not exists
    if not os.path.exists(prefix):
        os.makedirs(prefix)

    to_copy_directories = ['data', 'log', 'bin', 'web-dist', 'conf']
    dest_dir = os.path.join(prefix, 'tidb-foresight')
    mkdir_if_nonexists(dest_dir)

    # check tidb-foresight.toml
    # if exists, then copy it to $prefix/tidb-foresight
    if os.path.exists('tidb-foresight.toml'):
        os.system(
            'cp tidb-foresight.toml {}/tidb-foresight.toml'.format(dest_dir))

    for to_copy_directory in to_copy_directories:
        os.system("cp -r {} {}".format(
            to_copy_directory, os.path.join(dest_dir, to_copy_directory)))
    os.system("cp *.service {}".format(dest_dir))
    os.system("cp *.service /etc/systemd/system/".format(dest_dir))
    os.system("chmod 755 {}/*".format(dest_dir))
    os.system("chmod 755 {}/scripts/*".format(dest_dir))
