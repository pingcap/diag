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

HINT_CHECK_DICT = {
    "lsof -v": "please install lsof",
    "echo pong": "please install echo",
    # Below is a command for testing
    "adshk- du di da": "please do nothing...",
}


def check(result):
    if result['failed']:
        return 'failed'
    elif result['unreachable']:
        return 'unreachable'
    else:
        return 'success'


class TaskFactory:
    @staticmethod
    def whoami():
        return [dict(action=dict(module='shell', args='whoami'), become=True)]

    @staticmethod
    def ping():
        return [dict(action=dict(module='ping'))]

    @staticmethod
    def run_command(command):
        return [dict(action=dict(module='shell', args=command))]


class ResultCallback(CallbackBase):
    """
    A callback plugin used for performing an action as results come in

    If you want to collect all results into a single object for processing at
    the end of the execution, look into utilizing the ``json`` callback plugin
    or writing your own custom callback plugin
    """
    def __init__(self, *args, **kwargs):
        super(ResultCallback, self).__init__(*args, **kwargs)
        self.host_ok = {}
        self.host_unreachable = {}
        self.host_failed = {}

    @staticmethod
    def _load_host_name(result):
        """
        :param result: the result from hook.
        :return: the name of the result
        """
        return result._host.get_name()

    def v2_runner_on_unreachable(self, result):
        self.host_unreachable[ResultCallback._load_host_name(result)] = result

    def v2_runner_on_ok(self, result, *args, **kwargs):
        """
        Print a json representation of the result

        This method could store the result in an instance attribute for retrieval later
        """
        self.host_ok[ResultCallback._load_host_name(result)] = result

    def v2_runner_on_failed(self, result, *args, **kwargs):
        self.host_failed[ResultCallback._load_host_name(result)] = result


class AnsibleApi:
    """
    ansible hook and developing api: https://docs.ansible.com/ansible/latest/dev_guide/developing_api.html
    """
    def __init__(self, inv):
        """
        :param inv: input inv file
        """
        self.inv = inv

        Options = namedtuple('Options', [
            'connection', 'remote_user', 'ask_sudo_pass', 'verbosity',
            'ack_pass', 'module_path', 'forks', 'become', 'become_method',
            'become_user', 'check', 'listhosts', 'listtasks', 'listtags',
            'syntax', 'sudo_user', 'sudo', 'diff'
        ])

        self.ops = Options(connection='ssh',
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

        # load data from ansible
        self.loader = DataLoader()
        # the server should be connect with no password
        self.passwords = dict()
        self.results_callback = ResultCallback()
        self.inventory = InventoryManager(loader=self.loader, sources=self.inv)
        # variable manager takes care of merging all the different sources
        #  to give you a unified view of variables available in each context
        self.variable_manager = VariableManager(loader=self.loader,
                                                inventory=self.inventory)

    def run_ansible(self, host_list, task_list):
        """
        :param host_list: The list of ansible hosts
        :param task_list: The list of ansible tasks
        :return: hosts for success/failed/unreachable
        """
        play_source = {
            'name': "Ansible Play",
            'hosts': host_list,
            'gather_facts': 'no',
            'tasks': task_list
        }
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

        results_raw = dict()
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


# ClusterInfo = namedtuple('ClusterInfo', ['cluster_name', 'status', 'message', 'hosts'])
# AnsibleHost = namedtuple('AnsibleHost', ['status', 'ip', 'user', 'components', 'message'])


def check_exists_phase(required_commands, ip, inv_path):
    """
    :param required_commands: Dict[Command, Hint], Command and Hint are all `str`
    :param ip: the ip to testing
    :param inv_path:
    :return: a list of hinting.
    """
    hints = list()
    ansible_runner = AnsibleApi(inv_path)
    for command, hint in required_commands.iter():
        info = json.loads(
            ansible_runner.run_ansible([ip], TaskFactory.run_command(command)))
        if check(info) != 'success':
            hints.append(hint)
    return hint


def hostinfo(inv_path):
    """
    :param inv_path: an str to represent the inv file path
    :return:
        A dict represents the cluster info. for the whole class
            * cluster_name: str for whole cluster name
            * status: "exception" or "success"
            * message: the message from the
            * hosts: the message of the hosts
        And a host is Like:
            * status: "exception" or "success"
            * ip: ip of the host
            * user: the user of the system
            * components: Ansible components
            * message: system message
    """
    def check_node(ip, exist_hosts):
        """
        :param ip: the ip of the node to check
        :return: (bool, bool, List)
            Which represents the node (exists, if we can using sudo to access it)
        """
        _dict = {}
        _connect = []

        # This just check exists

        # exist = any(_ip in _info for (_info in hosts))
        exist = ip in [info.itervalues() for info in exist_hosts]

        # call the ping task, and using AnsibleApi to schedule it, the result will be pack into
        # `host_ok` and so on.
        # this task will set _connect.
        task_ping = TaskFactory.ping()
        # run ansible with inv file.
        run_ansible = AnsibleApi(inv_path)
        result_ping = json.loads(run_ansible.run_ansible([ip], task_ping))
        del run_ansible
        if 'unreachable' in result_ping:
            connect = [
                False, 'unreachable', 'Failed to connect to the host via ssh'
            ]
        elif 'failed' in result_ping:
            connect = [False, 'failed', result_ping['failed'][ip]]
        else:
            connect = [True, 'success']

        # call the whoami task, and using AnsibleApi to schedule it, the result will be pack into
        # `host_ok` and so on.
        # this task will set `sudo`.
        task_whoami = TaskFactory.whoami()

        result_whoami = json.loads(run_ansible.run_ansible([ip], task_whoami))

        sudo = result_whoami['success'] is not None
        return exist, sudo, connect

    def get_node_info(ip, deploy_dir, name):
        """
        :param ip: ip of node, an `str`
        :param deploy_dir: `str`
        :param name: Service name, an `str`
        :return: bool for ServiceStatus, enum for node info, [port, name]
        """
        _host = [ip]
        if name == 'pd':
            _command = 'cat ' + deploy_dir + '/scripts/run_pd.sh | grep "\--client-urls"'
            _task = TaskFactory.run_command(_command)
            runAnsible = AnsibleApi(inv_path)
            _info = json.loads(runAnsible.run_ansible(_host, _task))
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
            _task = TaskFactory.run_command(_command)
            runAnsible = AnsibleApi(inv_path)
            _info = json.loads(runAnsible.run_ansible(_host, _task))
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
            _task = TaskFactory.run_command(_command)
            runAnsible = AnsibleApi(inv_path)
            _info = json.loads(runAnsible.run_ansible(_host, _task))
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
            _task = TaskFactory.run_command(_command)
            runAnsible = AnsibleApi(inv_path)
            _info = json.loads(runAnsible.run_ansible(_host, _task))
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
                _task = TaskFactory.run_command(_command)
                runAnsible = AnsibleApi(inv_path)
                _info = json.loads(runAnsible.run_ansible(_host, _task))
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
                _task = TaskFactory.run_command(_command)
                runAnsible = AnsibleApi(inv_path)
                _info = json.loads(runAnsible.run_ansible(_host, _task))
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
    # set inventory manager and variable manager
    inv_manager = InventoryManager(loader=loader, sources=[inv_path])
    _vars = VariableManager(loader=loader, inventory=inv_manager)
    # TODO: make clear if this is Map[Servers, Script]
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

    # cluster_info = ClusterInfo(
    #     hosts=[],
    #     status=None,
    #     message=None,
    #     cluster_name=None
    # )
    cluster_info = {}
    hosts = []

    # We merge a 'magic' var 'groups' with group name keys and hostname
    # list values into every host variable set.
    all_group = inv_manager.get_groups_dict()
    all_group.pop('all')

    for group, _host_list in all_group.iteritems():
        if not _host_list:
            continue
        for _host in _host_list:
            hostvars = _vars.get_vars(host=inv_manager.get_host(
                hostname=str(_host)))  # get all variables for one node
            _deploy_dir = hostvars['deploy_dir']
            _cluster_name = hostvars['cluster_name']
            _tidb_version = hostvars['tidb_version']
            _ansible_user = hostvars['ansible_user']
            # initialize cluster_name and tidb_version
            if 'cluster_name' not in cluster_info:
                cluster_info['cluster_name'] = _cluster_name
            if 'tidb_version' not in cluster_info:
                cluster_info['tidb_version'] = _tidb_version
            _ip = hostvars['ansible_host'] if 'ansible_host' in hostvars else \
                (hostvars['ansible_ssh_host'] if 'ansible_ssh_host' in hostvars else
                 hostvars['inventory_hostname'])
            # check with ansible ping sdk and ansible `whoami`
            _ip_exist, _enable_sudo, _enable_connect = check_node(_ip, hosts)

            if not _ip_exist:
                # adding ip to hist list
                _host_dict = {}
                _host_dict['ip'] = _ip
                _host_dict['user'] = _ansible_user
                _host_dict['components'] = []

                if _enable_connect[0]:
                    # can connect to the server
                    _host_dict['status'] = 'success'
                    _host_dict['message'] = ''
                    _host_dict['enable_sudo'] = _enable_sudo
                    _host_dict['hints'] = check_exists_phase(
                        HINT_CHECK_DICT, _ip, inv_path)
                else:
                    # exception
                    _host_dict['status'] = 'exception'
                    _host_dict['message'] = _enable_connect[2]
                hosts.append(_host_dict)

            _status, _type, _info = get_node_info(_ip, _deploy_dir,
                                                  server_group[group])
            for _index_id in range(len(hosts)):
                if hosts[_index_id]['ip'] == _ip:
                    if hosts[_index_id]['status'] == 'exception':
                        break
                    if group != 'monitoring_servers' and group != 'monitored_servers':
                        _dict1 = {}
                        if not _status and _type == 'get_info':
                            _dict1['name'] = _info[1]
                            _dict1['status'] = 'exception'
                            _dict1['message'] = _info[0][_host]
                            _dict1['deploy_dir'] = _deploy_dir
                        elif _status and _type == 'get_info':
                            _dict1['name'] = _info[1]
                            _dict1['status'] = 'success'
                            if group == 'tidb_servers':
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
    errmessage = []
    for hostinfo in hosts:
        if hostinfo['ip'] in errmessage:
            continue
        if hostinfo['status'] == 'exception':
            errmessage.append(hostinfo['ip'])
            continue
        else:
            for _serverinfo in hostinfo['components']:
                if _serverinfo['status'] == 'exception':
                    errmessage.append(hostinfo['ip'])
                    break
    if errmessage:
        cluster_info['status'] = 'exception'
        cluster_info['message'] = 'Fail list: ' + str(errmessage)
    else:
        cluster_info['status'] = 'success'
        cluster_info['message'] = ''
    return cluster_info


if __name__ == '__main__':
    if len(sys.argv) <= 1:
        raise RuntimeError("Too few arguments")
    inventory = sys.argv[1]
    result = hostinfo(inventory)
    print json.dumps(result, indent=4)
