#!/usr/bin/env python
# -*- coding=utf-8 -*-

import sys
import os
from string import Template

if __name__ == '__main__':
    prefix = sys.argv[1]
    # move all to prefix
    # if prefix is a subdirectory of current dir, it will fail
    print(os.path.realpath(prefix), os.path.realpath('.'))
    if os.path.realpath('.') in os.path.realpath(prefix):
        print('prefix is a subdirectory of src files, cannot work')
        exit(1)
    arguments_dict = {'prefix': prefix}
    service_list = [('prometheus', 9527), ('influxd', 9528),
                    ('foresight', 9529)]

    # create and copy files for them

    src_prefix = 'templates'

    for service_name, number in service_list:
        with open(
                'templates/{}.template.service'.format(service_name),
                mode='r') as template_s:
            src = Template(template_s.read())
            d = {'prefix': prefix}

            with open('{}-{}.service'.format(service_name, number), 'w+') as fw:
                fw.write(src.safe_substitute(arguments_dict))

    # need to check and create if prefix not exists
    if not os.path.exists(prefix):
        os.makedirs(prefix)

    os.system("cp -r * {}".format(prefix))
    os.system("mv {}/*.service /etc/systemd/system/".format(prefix))
    os.system("chmod 755 {}/*".format(prefix))
