#!/usr/bin/env python

import sys
import os
import subprocess
import platform


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
    manager = package_manager()
    if manager == 'yum':
        os.system(
            'curl --silent --location https://rpm.nodesource.com/setup_10.x | sudo bash -'
        )
        os.system(
            'curl --silent --location https://dl.yarnpkg.com/rpm/yarn.repo | sudo tee /etc/yum.repos.d/yarn.repo'
        )
    elif manager == 'brew':
        os.system("brew install yarn")
    else:
        os.system(
            'curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -'
        )
        os.system(
            'echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list'
        )
        os.system('apt-get update')

    os.system('{} install -y nodejs yarn'.format(manager))

    download_prefix = 'http://fileserver.pingcap.net/download/foresight/'
    if 'http' not in sys.argv[1]:
        download_prefix = None
    for to_install in sys.argv[2:]:
        if not os.path.exists('bin/' + to_install):
            to_execute = 'wget http://fileserver.pingcap.net/download/foresight/{} --directory-prefix=./bin'.format(
                to_install)
            print to_execute
            os.system(to_execute)
