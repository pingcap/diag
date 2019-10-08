#!/usr/bin/env python
# -*- coding=utf-8 -*-

import sys
import os
from string import Template

if __name__ == '__main__':
    prefix = sys.argv[1]
    arguments_dict = {'prefix': prefix}
    service_list = [('prometheus', 9529), ('influxd', 9528),
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
    print('nmsl')
    # need to check and create if dest_prefix not exists
    dest_prefix = '/opt/tidb/tidb-foresight'
    if not os.path.exists(dest_prefix):
        os.makedirs(dest_prefix)

    # move all to dest_prefix
    # if dest_prefix is a subdirectory of current dir, it will fail
    print(os.path.realpath(dest_prefix), os.path.realpath('.'))
    if os.path.realpath('.') in os.path.realpath(dest_prefix):
        print('dest_prefix is a subdirectory of src files, cannot work')
        exit(1)
    os.system("cp -r * {}".format(dest_prefix))
    os.system("mv {}/*.service /etc/systemd/system/".format(dest_prefix))
    os.system("chmod 755 {}/*".format(dest_prefix))

    # start services
    os.system("systemctl daemon-reload")
    for service_name, number in service_list:
        os.system("systemctl {}-{}".format(service_name, number))
