#!/usr/bin/env python

import sys
import os

if __name__ == '__main__':
    download_prefix = 'http://fileserver.pingcap.net/download/foresight/'
    if 'http' not in sys.argv[1]:
        download_prefix = None
    for to_install in sys.argv[2:]:
        if not os.path.exists('bin/' + to_install):
            to_execute = 'wget http://fileserver.pingcap.net/download/foresight/{} --directory-prefix=./bin'.format(
                to_install)
            print to_execute
            os.system(to_execute)
