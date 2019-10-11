#!/usr/bin/env python

import sys
import os
import subprocess


def package_manager():
    script = "awk -F= '/^NAME/{print $2}' /etc/os-release"
    try:
        edition = subprocess.check_output(script, shell=True).strip().lower()
    except OSError as e:
        print('cannot run script: {}'.format(script))
        raise e
    mapper = {'ubuntu': 'apt-get', 'centos': 'yum'}
    for k, v in mapper.iteritems():
        if k in edition:
            return v
    raise ValueError("don't know the package manager for {}".format(edition))


if __name__ == '__main__':

    os.system('{} install -y graphviz perf rsync golang'.format(
        package_manager()))

    download_prefix = 'http://fileserver.pingcap.net/download/foresight/'
    if 'http' not in sys.argv[1]:
        download_prefix = None
    for to_install in sys.argv[2:]:
        if not os.path.exists('bin/' + to_install):
            to_execute = 'wget http://fileserver.pingcap.net/download/foresight/{} --directory-prefix=./bin'.format(
                to_install)
            print to_execute
            os.system(to_execute)
