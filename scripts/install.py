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
import platform
from string import Template


def mkdir_if_nonexists(path):
    """
    Create and empty directory for path.
    path: `str` for path of the directory to create.
    """
    os.system("mkdir -p {}".format(path))


def generate_service(prefix, prometheus_port, influxd_port, foresight_port,
                     user):
    """
    Remove all history service files and generate directory for config.
    """
    # remove all historical service files.
    os.system("rm *.service")
    # mkdir_if_nonexists('conf')
    arguments_dict = {
        'prefix': prefix,
        'prometheus_port': prometheus_port,
        'influxd_port': influxd_port,
        'foresight_port': foresight_port,
        'user': user,
    }
    service_list = [('prometheus', prometheus_port), ('influxd', influxd_port),
                    ('foresight', foresight_port)]

    # create and copy files for them
    for service_name, number in service_list:
        with open(
                'templates/{}.template.service'.format(service_name),
                mode='r') as template_s:
            src = Template(template_s.read())
            # generate prefix for config
            with open('{}-{}.service'.format(service_name, number), 'w+') as fw:
                fw.write(src.safe_substitute(arguments_dict))


def generate_conf(prefix, prometheus_port, influxd_port, foresight_port, user):
    mkdir_if_nonexists('conf')
    arguments_dict = {
        'prefix': prefix,
        'prometheus_port': prometheus_port,
        'influxd_port': influxd_port,
        'foresight_port': foresight_port,
        'user': user
    }
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


def chmod_path(abs_path):
    """
    path: an `str` for the path
    """
    pos = abs_path.find('/')
    while pos != -1:
        current_path = abs_path[:pos + 1]
        os.system("chmod go+x {}".format(current_path))
        pos = abs_path.find('/', pos + 1)


def package_manager():

    def linux_distribution():
        """
        return: `str` for current linux distribution
        """
        try:
            return ' '.join(platform.linux_distribution())
        except:
            return "N/A"

    validating_list = (linux_distribution(), platform.system(),
                       platform.version())

    # https://docs.python.org/3/library/platform.html?highlight=uname#platform.version
    for version in validating_list:
        edition = version.strip().lower()
        mapper = {'ubuntu': 'apt-get', 'centos': 'yum', 'darwin': 'brew'}
        for k, v in mapper.iteritems():
            if k in edition:
                return v
    print('not centos/ubuntu/osx, use apt-get as package manager.')
    return 'apt-get'


if __name__ == '__main__':
    # install the requirements
    os.system('{} install -y graphviz perf rsync golang'.format(
        package_manager()))

    prefix = os.path.abspath(sys.argv[1])
    user = sys.argv[2]
    (foresight_port, influxd_port,
     prometheus_port) = map(validate_int, sys.argv[3:])
    # move all to prefix
    # if prefix is a subdirectory of current dir, it will fail
    print(os.path.realpath(prefix), os.path.realpath('.'))
    if os.path.realpath('.') in os.path.realpath(prefix):
        print('prefix is a subdirectory of src files, cannot work')
        exit(1)

    generate_service(prefix, prometheus_port, influxd_port, foresight_port,
                     user)
    generate_conf(prefix, prometheus_port, influxd_port, foresight_port, user)

    # final stage for copying files
    # need to check and create if prefix not exists
    if not os.path.exists(prefix):
        os.makedirs(prefix)

    to_copy_directories = ['bin', 'web-dist', 'conf']
    # 最终的 $prefix/tidb-foresight 文件
    dest_dir = os.path.join(prefix, 'tidb-foresight')
    # override
    mkdir_if_nonexists(dest_dir)

    directories = [
        'data', 'data/influxdb', 'data/prometheus', 'log', 'log/influxdb',
        'log/prometheus'
    ]
    for to_create_dir in directories:
        # create them in dest_dir
        to_create_dir = os.path.join(dest_dir, to_create_dir)
        mkdir_if_nonexists(to_create_dir)

    # check tidb-foresight.toml
    # if exists, then copy it to $prefix/tidb-foresight
    if os.path.exists('tidb-foresight.toml'):
        os.system(
            "'cp' -f tidb-foresight.toml {}/tidb-foresight.toml".format(
                dest_dir))

    for to_copy_directory in to_copy_directories:
        os.system("'cp' -rf {} {}".format(to_copy_directory, dest_dir))
    # rename web-dist to web
    print("mv {dest_dir}/web-dist {dest_dir}/web".format(dest_dir=dest_dir))
    os.system(
        "rm -rf {dest_dir}/web {dest_dir}/web-dist && 'cp' -rf web-dist {dest_dir}/ && mv {dest_dir}/web-dist {dest_dir}/web"
        .format(dest_dir=dest_dir))

    os.system("'cp' -rf *.service {}".format(dest_dir))
    os.system("'cp' -rf *.service /etc/systemd/system/")
    os.system('chown {} -R $prefix'.format(user))
    chmod_path(dest_dir)

    os.system("chmod a+x {}/*".format(dest_dir))
    os.system("chmod a+x {}/bin/*".format(dest_dir))
