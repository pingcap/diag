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
    print('not centos, use apt-get as package manager.')
    return 'apt-get'


if __name__ == '__main__':
    manager = package_manager()
    if manager == 'yum':
        os.system(
            'curl --silent --location https://dl.yarnpkg.com/rpm/yarn.repo | sudo tee /etc/yum.repos.d/yarn.repo'
        )
    else:
        os.system(
            'curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -'
        )
        os.system(
            'echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list'
        )
        os.system('apt-get update')
    os.system('{} install -y graphviz perf rsync golang yarn'.format(manager))

    download_prefix = 'http://fileserver.pingcap.net/download/foresight/'
    if 'http' not in sys.argv[1]:
        download_prefix = None
    for to_install in sys.argv[2:]:
        if not os.path.exists('bin/' + to_install):
            to_execute = 'wget http://fileserver.pingcap.net/download/foresight/{} --directory-prefix=./bin'.format(
                to_install)
            print to_execute
            os.system(to_execute)
