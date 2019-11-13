#!/usr/bin/env python2
# coding: utf-8

import re
import sys
import json
import shutil
from collections import namedtuple

import ansible.constants as C
from ansible.playbook.play import Play
from ansible.executor.task_queue_manager import TaskQueueManager
from ansible.plugins.callback import CallbackBase
from ansible.parsing.dataloader import DataLoader
from ansible.inventory.manager import InventoryManager
from ansible.vars.manager import VariableManager


class ResultCallback(CallbackBase):

    def __init__(self, *args, **kwargs):
        super(ResultCallback, self).__init__(*args, **kwargs)
        self.host_ok = {}
        self.host_unreachable = {}
        self.host_failed = {}

    def v2_runner_on_unreachable(self, result):
        self.host_unreachable[result._host.get_name()] = result

    def v2_runner_on_ok(self, result, *args, **kwargs):
        self.host_ok[result._host.get_name()] = result

    def v2_runner_on_failed(self, result, *args, **kwargs):
        self.host_failed[result._host.get_name()] = result


class AnsibleApi(object):

    def __init__(self, inv):
        self.inv = inv
        self.Options = namedtuple('Options', [
            'connection', 'remote_user', 'ask_sudo_pass', 'verbosity',
            'ack_pass', 'module_path', 'forks', 'become', 'become_method',
            'become_user', 'check', 'listhosts', 'listtasks', 'listtags',
            'syntax', 'sudo_user', 'sudo', 'diff'
        ])

        self.ops = self.Options(
            connection='ssh',
            remote_user=None,
            ack_pass=None,
            sudo_user=None,
            forks=5,
            sudo=None,
            ask_sudo_pass=False,
            verbosity=5,
            module_path=None,
            become=None,
            become_method='sudo',
            become_user='root',
            check=False,
            diff=False,
            listhosts=None,
            listtasks=None,
            listtags=None,
            syntax=None)

        self.loader = DataLoader()
        self.passwords = dict()
        self.results_callback = ResultCallback()
        self.inventory = InventoryManager(loader=self.loader, sources=self.inv)
        self.variable_manager = VariableManager(
            loader=self.loader, inventory=self.inventory)

    def runansible(self, host_list, task_list):

        play_source = dict(
            name="Ansible Play",
            hosts=host_list,
            gather_facts='no',
            tasks=task_list)
        play = Play().load(
            play_source,
            variable_manager=self.variable_manager,
            loader=self.loader)

        tqm = None
        try:
            tqm = TaskQueueManager(
                inventory=self.inventory,
                variable_manager=self.variable_manager,
                loader=self.loader,
                options=self.ops,
                passwords=self.passwords,
                stdout_callback=self.results_callback,
                run_additional_callbacks=C.DEFAULT_LOAD_CALLBACK_PLUGINS,
                run_tree=False,
            )
            result = tqm.run(play)
        finally:
            if tqm is not None:
                tqm.cleanup()
            shutil.rmtree(C.DEFAULT_LOCAL_TMP, True)

        results_raw = {}
        results_raw['success'] = {}
        results_raw['failed'] = {}
        results_raw['unreachable'] = {}

        for host, result in self.results_callback.host_ok.items():
            results_raw['success'][host] = result._result

        for host, result in self.results_callback.host_failed.items():
            results_raw['failed'][host] = result._result['stderr']

        for host, result in self.results_callback.host_unreachable.items():
            results_raw['unreachable'][host] = result._result['msg']

        return json.dumps(results_raw, indent=4)


def hostinfo(inv):

    def check_node(ip):
        _exist = False
        _dict = {}
        _connect = []

        if hosts:
            for _info in hosts:
                if _ip in _info.itervalues():
                    _exist = True
                    break

        _sudo = False
        _task1 = [dict(action=dict(module='ping'))]
        runAnsible = AnsibleApi(inv)
        _result1 = json.loads(runAnsible.runansible([ip], _task1))
        del runAnsible
        if _result1['unreachable']:
            _connect = [
                False, 'unreachable', 'Failed to connect to the host via ssh'
            ]
        elif _result1['failed']:
            _connect = [False, 'failed', _result1['failed'][ip]]
        else:
            _connect = [True, 'success']

        _task2 = [dict(action=dict(module='shell', args='whoami'), become=True)]
        runAnsible = AnsibleApi(inv)
        _result2 = json.loads(runAnsible.runansible([ip], _task2))
        del runAnsible
        if _result2['success']:
            _sudo = True

        return _exist, _sudo, _connect

    def check(result):
        if result['failed']:
            return 'failed'
        elif result['unreachable']:
            return 'unreachable'
        else:
            return 'success'

    def get_node_info(ip, deploy_dir, name):
        _host = [ip]
        if name == 'pd':
            _command = 'cat ' + deploy_dir + '/scripts/run_pd.sh | grep "\--client-urls"'
            _task = [dict(action=dict(module='shell', args=_command))]
            runAnsible = AnsibleApi(inv)
            _info = json.loads(runAnsible.runansible(_host, _task))
            del runAnsible
            ok = check(_info)
            if ok == 'success':
                _port = re.search(
                    "([0-9]+)\"",
                    _info['success'][ip]['stdout_lines'][0]).group(1)
                return True, 'get_info', [_port, name]
            else:
                return False, 'get_info', [_info[ok], name]
        elif name == 'tidb':
            _command = 'cat ' + deploy_dir + '/scripts/run_tidb.sh | grep -E "\-P|--status"'
            _task = [dict(action=dict(module='shell', args=_command))]
            runAnsible = AnsibleApi(inv)
            _info = json.loads(runAnsible.runansible(_host, _task))
            del runAnsible
            ok = check(_info)
            if ok == 'success':
                _port = re.search(
                    "([0-9]+)",
                    _info['success'][ip]['stdout_lines'][0]).group(1)
                _status_port = re.search(
                    "([0-9]+)\"",
                    _info['success'][ip]['stdout_lines'][1]).group(1)
                return True, 'get_info', [[_port, _status_port], name]
            else:
                return False, 'get_info', [_info[ok], name]
        elif name == 'tikv':
            _command = 'cat ' + deploy_dir + '/scripts/run_tikv.sh | grep "\--addr"'
            _task = [dict(action=dict(module='shell', args=_command))]
            runAnsible = AnsibleApi(inv)
            _info = json.loads(runAnsible.runansible(_host, _task))
            del runAnsible
            ok = check(_info)
            if ok == 'success':
                _port = re.search(
                    "([0-9]+)\"",
                    _info['success'][ip]['stdout_lines'][0]).group(1)
                return True, 'get_info', [_port, name]
            else:
                return False, 'get_info', [_info[ok], name]
        elif name == 'grafana':
            _command = 'cat ' + deploy_dir + '/opt/grafana/conf/grafana.ini | grep "^http_port"'
            _task = [dict(action=dict(module='shell', args=_command))]
            runAnsible = AnsibleApi(inv)
            _info = json.loads(runAnsible.runansible(_host, _task))
            del runAnsible
            ok = check(_info)
            if ok == 'success':
                _port = re.search(
                    "([0-9]+)",
                    _info['success'][ip]['stdout_lines'][0]).group(1)
                return True, 'get_info', [_port, name]
            else:
                return False, 'get_info', [_info[ok], name]
        elif name == 'monitoring':
            _check = []
            _result = []
            for _server in ['prometheus', 'pushgateway']:
                _command = 'cat ' + deploy_dir + '/scripts/run_' + _server + '\.sh | grep "\--web\.listen-address"'
                _task = [dict(action=dict(module='shell', args=_command))]
                runAnsible = AnsibleApi(inv)
                _info = json.loads(runAnsible.runansible(_host, _task))
                del runAnsible
                ok = check(_info)
                if ok == 'success':
                    _check.append(True)
                    _result.append([
                        re.search(
                            "([0-9]+)\"",
                            _info['success'][ip]['stdout_lines'][0]).group(1),
                        _server
                    ])
                else:
                    _check.append(False)
                    _result.append([_info[ok], _server])
            return _check, 'get_info', _result
        elif name == 'monitored':
            _check = []
            _result = []
            for _server in ['node_exporter', 'blackbox_exporter']:
                _command = 'cat ' + deploy_dir + '/scripts/run_' + _server + '\.sh | grep "\--web\.listen-address"'
                _task = [dict(action=dict(module='shell', args=_command))]
                runAnsible = AnsibleApi(inv)
                _info = json.loads(runAnsible.runansible(_host, _task))
                del runAnsible
                ok = check(_info)
                if ok == 'success':
                    _check.append(True)
                    _result.append([
                        re.search(
                            "([0-9]+)\"",
                            _info['success'][ip]['stdout_lines'][0]).group(1),
                        _server
                    ])
                else:
                    _check.append(False)
                    _result.append([_info[ok], _server])
            return _check, 'get_info', _result
        else:
            return False, 'other', name

    loader = DataLoader()
    _inv = InventoryManager(loader=loader, sources=[inv])
    _vars = VariableManager(loader=loader, inventory=_inv)
    server_group = {
        'pd_servers': 'pd',
        'tidb_servers': 'tidb',
        'tikv_servers': 'tikv',
        'monitoring_servers': 'monitoring',
        'monitored_servers': 'monitored',
        'alertmanager_servers': 'alertmanager',
        'drainer_servers': 'drainer',
        'pump_servers': 'pump',
        'spark_master': 'spark_master',
        'spark_slaves': 'spark_slave',
        'lightning_server': 'lightning',
        'importer_server': 'importer',
        'kafka_exporter_servers': 'kafka_exporter',
        'grafana_servers': 'grafana'
    }

    cluster_info = {}
    hosts = []

    _all_group = _inv.get_groups_dict()
    _all_group.pop('all')
    for _group, _host_list in _all_group.iteritems():
        if not _host_list:
            continue
        for _host in _host_list:
            _hostvars = _vars.get_vars(host=_inv.get_host(
                hostname=str(_host)))  # get all variables for one node
            _deploy_dir = _hostvars['deploy_dir']
            _cluster_name = _hostvars['cluster_name']
            _tidb_version = _hostvars['tidb_version']
            _ansible_user = _hostvars['ansible_user']
            if 'cluster_name' not in cluster_info:
                cluster_info['cluster_name'] = _cluster_name
            if 'tidb_version' not in cluster_info:
                cluster_info['tidb_version'] = _tidb_version
            _ip = _hostvars['ansible_host'] if 'ansible_host' in _hostvars else \
                (_hostvars['ansible_ssh_host'] if 'ansible_ssh_host' in _hostvars else
                 _hostvars['inventory_hostname'])
            _ip_exist, _enable_sudo, _enable_connect = check_node(_ip)
            if not _ip_exist:
                _host_dict = {}
                _host_dict['ip'] = _ip
                _host_dict['user'] = _ansible_user
                _host_dict['components'] = []
                if _enable_connect[0]:
                    _host_dict['status'] = 'success'
                    _host_dict['message'] = ''
                    _host_dict['enable_sudo'] = _enable_sudo
                else:
                    _host_dict['status'] = 'exception'
                    _host_dict['message'] = _enable_connect[2]
                hosts.append(_host_dict)

            _status, _type, _info = get_node_info(_ip, _deploy_dir,
                                                  server_group[_group])
            for _index_id in range(len(hosts)):
                if hosts[_index_id]['ip'] == _ip:
                    if hosts[_index_id]['status'] == 'exception':
                        break
                    if _group != 'monitoring_servers' and _group != 'monitored_servers':
                        _dict1 = {}
                        if not _status and _type == 'get_info':
                            _dict1['name'] = _info[1]
                            _dict1['status'] = 'exception'
                            _dict1['message'] = _info[0][_host]
                            _dict1['deploy_dir'] = _deploy_dir
                        elif _status and _type == 'get_info':
                            _dict1['name'] = _info[1]
                            _dict1['status'] = 'success'
                            if _group == 'tidb_servers':
                                _dict1['port'] = _info[0][0]
                                _dict1['status_port'] = _info[0][1]
                            else:
                                _dict1['port'] = _info[0]
                            _dict1['deploy_dir'] = _deploy_dir
                        if _dict1:
                            hosts[_index_id]['components'].append(_dict1)
                    else:
                        for _indexid in range(2):
                            _dict2 = {}
                            if not _status[_indexid]:
                                _dict2['name'] = _info[_indexid][1]
                                _dict2['status'] = 'exception'
                                _dict2['message'] = _info[_indexid][0][_host]
                                _dict2['deploy_dir'] = _deploy_dir
                            else:
                                _dict2['name'] = _info[_indexid][1]
                                _dict2['status'] = 'success'
                                _dict2['port'] = _info[_indexid][0]
                                _dict2['deploy_dir'] = _deploy_dir
                            hosts[_index_id]['components'].append(_dict2)
    cluster_info['hosts'] = hosts
    _errmessage = []
    for _hostinfo in hosts:
        if _hostinfo['ip'] in _errmessage:
            continue
        if _hostinfo['status'] == 'exception':
            _errmessage.append(_hostinfo['ip'])
            continue
        else:
            for _serverinfo in _hostinfo['components']:
                if _serverinfo['status'] == 'exception':
                    _errmessage.append(_hostinfo['ip'])
                    break
    if _errmessage:
        cluster_info['status'] = 'exception'
        cluster_info['message'] = 'Fail list: ' + str(_errmessage)
    else:
        cluster_info['status'] = 'success'
        cluster_info['message'] = ''
    return cluster_info


if __name__ == '__main__':
    inventory = sys.argv[1]
    result = hostinfo(inventory)
    print json.dumps(result, indent=4)
