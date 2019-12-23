#!/usr/bin/env python2
# coding: utf-8

import re
import sys
import json
import shutil
from multiprocessing import Process, Pool
from collections import namedtuple
from collections import defaultdict

import ansible.constants as C
from ansible.playbook.play import Play
from ansible.executor.task_queue_manager import TaskQueueManager
from ansible.plugins.callback import CallbackBase
from ansible.parsing.dataloader import DataLoader
from ansible.inventory.manager import InventoryManager
from ansible.vars.manager import VariableManager

HINT_CHECK_DICT = {
    "lsof -v": "`lsof` not exists on your machine, please install it",
    "netstat --version":
    "`netstat` not exists on your machine, please install it",
    "ntpq --version": "`netq` not exists on your machine, please install it"
}


def lock_manager(lock):
    lock.acquire()
    try:
        yield None
    finally:
        lock.release()


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
    """
    * Warning: This should be run only one time.
    """
    def __init__(self, inv):
        # used is a flag for if this method is used.
        # If it's used, the AnsibleApi will raise an exception.
        self.used = False
        self.inv = inv
        self.Options = namedtuple('Options', [
            'connection', 'remote_user', 'ask_sudo_pass', 'verbosity',
            'ack_pass', 'module_path', 'forks', 'become', 'become_method',
            'become_user', 'check', 'listhosts', 'listtasks', 'listtags',
            'syntax', 'sudo_user', 'sudo', 'diff'
        ])

        self.ops = self.Options(connection='ssh',
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
        self.variable_manager = VariableManager(loader=self.loader,
                                                inventory=self.inventory)

    def _runansible_impl(self, host_list, task_list):
        play_source = dict(name="Ansible Play",
                           hosts=host_list,
                           gather_facts='no',
                           tasks=task_list)
        play = Play().load(play_source,
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
            tqm.run(play)
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

    def runansible(self, host_list, task_list):
        if self.used:
            raise RuntimeError(
                "method `runansible is used, please not call it again`")
        else:
            self.used = True

        max_try, tried = 5, 0
        while True:
            try:
                tried += 1
                return self._runansible_impl(host_list, task_list)
            except (KeyError, TypeError):
                if tried >= max_try:
                    raise
                continue


def check(result):
    """
    check the result object return in `runansible`.
    :param result:
    :return: 'failed'/'unreachable'/'success'
    """
    if result['failed']:
        return 'failed'
    elif result['unreachable']:
        return 'unreachable'
    else:
        return 'success'


def check_exists_phase(required_commands, ip, inv_path):
    """
    :param required_commands: Dict[Command, Hint], Command and Hint are all `str`
    :param ip: the ip to testing
    :param inv_path:
    :return: a list of hinting.
    """
    hints = list()
    for command, hint in required_commands.items():
        command = "source /etc/profile && source ~/.bash_profile && source ~/.bashrc && " + command
        ansible_runner = AnsibleApi(inv_path)
        info = json.loads(
            ansible_runner.runansible(
                [ip], [dict(action=dict(module='shell', args=command))]))
        if check(info) != 'success':
            hints.append(hint)
    return hints


class TaskFactory:
    def __init__(self):
        # The class should not have an instance.
        raise RuntimeError("Please not create instance for TaskFactory.")

    @staticmethod
    def whoami():
        return [dict(action=dict(module='shell', args='whoami'), become=True)]

    @staticmethod
    def ping():
        return [dict(action=dict(module='ping'))]

    @staticmethod
    def run_command(command):
        return [dict(action=dict(module='shell', args=command))]


def get_node_info(ip, deploy_dir, name, inv):
    _host = [ip]
    if name == 'pd':
        _command = 'cat ' + deploy_dir + '/scripts/run_pd.sh | grep "\--client-urls"'
        _task = [dict(action=dict(module='shell', args=_command))]
        runAnsible = AnsibleApi(inv)
        _info = json.loads(runAnsible.runansible(_host, _task))
        del runAnsible
        ok = check(_info)
        if ok == 'success':
            _port = re.search("([0-9]+)\"",
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
            _port = re.search("([0-9]+)",
                              _info['success'][ip]['stdout_lines'][0]).group(1)
            _status_port = re.search(
                "([0-9]+)\"", _info['success'][ip]['stdout_lines'][1]).group(1)
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
            _port = re.search("([0-9]+)\"",
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
            _port = re.search("([0-9]+)",
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


def run_task(ip, deploy_dir, group, inv, server_group, current_host):
    _status, _type, _info = get_node_info(ip, deploy_dir, server_group[group],
                                          inv)

    if current_host['status'] == 'exception':
        return
    if group != 'monitoring_servers' and group != 'monitored_servers':
        _dict1 = {}
        if not _status and _type == 'get_info':
            _dict1['name'] = _info[1]
            _dict1['status'] = 'exception'
            # TODO: This part is removed, please change it to right.
            # _dict1['message'] = _info[0][_host]
            # _dict1['message'] = _info[0][host]
            _dict1['deploy_dir'] = deploy_dir
        elif _status and _type == 'get_info':
            _dict1['name'] = _info[1]
            _dict1['status'] = 'success'
            if group == 'tidb_servers':
                _dict1['port'] = _info[0][0]
                _dict1['status_port'] = _info[0][1]
            else:
                _dict1['port'] = _info[0]
            _dict1['deploy_dir'] = deploy_dir
        if _dict1:
            current_host['components'].append(_dict1)
    else:
        for _indexid in range(2):
            _dict2 = {}
            if not _status[_indexid]:
                _dict2['name'] = _info[_indexid][1]
                _dict2['status'] = 'exception'
                _dict2['message'] = _info[_indexid][0][_host]
                _dict2['deploy_dir'] = deploy_dir
            else:
                _dict2['name'] = _info[_indexid][1]
                _dict2['status'] = 'success'
                _dict2['port'] = _info[_indexid][0]
                _dict2['deploy_dir'] = deploy_dir
            current_host['components'].append(_dict2)


def check_node_impl(ip, inv):
    """
    check_node check the ip, and return the node information
    let me see the detail info for other.
    :param ip:
    :return:
    """
    _exist = False
    _dict = {}
    _connect = []

    _sudo = False
    _task1 = TaskFactory.ping()
    run = AnsibleApi(inv)
    _result1 = json.loads(run.runansible([ip], _task1))
    if _result1['unreachable']:
        _connect = [
            False, 'unreachable', 'Failed to connect to the host via ssh'
        ]
    elif _result1['failed']:
        _connect = [False, 'failed', _result1['failed'][ip]]
    else:
        _connect = [True, 'success']

    _task2 = TaskFactory.whoami()
    run = AnsibleApi(inv)
    _result2 = json.loads(run.runansible([ip], _task2))
    if _result2['success']:
        _sudo = True

    return _exist, _sudo, _connect


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


def inner_func(node_ip, datalist, inv):
    """
    inner_func will capture hosts outside.
    """
    check_node = check_node_impl

    ip_exist, enable_sudo, enable_connect = check_node(node_ip, inv)
    assert ip_exist is False
    host_dict = {
        'ip': node_ip,
        'user': datalist[0][0],
        'components': [],
    }
    if enable_connect[0]:
        host_dict.update({
            'message':
            '',
            'enable_sudo':
            enable_sudo,
            'hints':
            check_exists_phase(HINT_CHECK_DICT, node_ip, inv),
            'status':
            'success',
        })
    else:
        host_dict.update({'status': 'exception', 'message': enable_connect[2]})
    # TODO Using this after making clear the logic.
    # GLOBAL_POOL.map(
    #     run_task,
    #     ((node_ip, configs[1], configs[2], inv, server_group, host_dict)
    #      for configs in datalist))
    for configs in datalist:
        run_task(node_ip, configs[1], configs[2], inv, server_group, host_dict)

    # now return the host_dict
    return host_dict


def inner_func_wrapper(tup):
    return inner_func(tup[0], tup[1], tup[2])


# global process pool
GLOBAL_POOL = Pool(4)


def hostinfo(inv):

    loader = DataLoader()
    inv_manager = InventoryManager(loader=loader, sources=[inv])
    vars_manager = VariableManager(loader=loader, inventory=inv_manager)

    cluster_info = {}
    hosts = []

    _all_group = inv_manager.get_groups_dict()
    _all_group.pop('all')

    # inv, server_group is global
    # it stores a set for ip -> [(ansible_user, deploy_dir, group)]
    node_map = defaultdict(list, dict())
    for _group, _host_list in _all_group.iteritems():
        if not _host_list:
            continue
        for _host in _host_list:
            _hostvars = vars_manager.get_vars(host=inv_manager.get_host(
                hostname=str(_host)))  # get all variables for one node
            _deploy_dir = _hostvars['deploy_dir']
            _cluster_name = _hostvars['cluster_name']
            _tidb_version = _hostvars['tidb_version']
            _ansible_user = _hostvars['ansible_user']
            # init cluster info
            if 'cluster_name' not in cluster_info:
                cluster_info['cluster_name'] = _cluster_name
            if 'tidb_version' not in cluster_info:
                cluster_info['tidb_version'] = _tidb_version
            _ip = _hostvars['ansible_host'] if 'ansible_host' in _hostvars else \
                (_hostvars['ansible_ssh_host'] if 'ansible_ssh_host' in _hostvars else
                 _hostvars['inventory_hostname'])
            node_map[_ip].append((_ansible_user, _deploy_dir, _group))

    # to_inserts = GLOBAL_POOL.map(inner_func_wrapper,
    #                              [(ip, datalist, inv)
    #                               for ip, datalist in node_map.iteritems()])

    res_list = []
    for ip, datalist in node_map.iteritems():
        res_list.append(GLOBAL_POOL.apply_async(inner_func, args=(ip, datalist, inv)))
    for res in res_list:
        hosts.append(res.get())

    # waiting for all task done.
    # for task in task_thread_list:
    #     task.join()

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
