package syncer

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	. "github.com/pingcap/check"
)

func Test(t *testing.T) { TestingT(t) }

type SyncTestSuit struct {
	uuid    string
	cluster cluster
}

var _ = Suite(&SyncTestSuit{})

var topologyJsonExample = `
{
    "cluster_name": "test-cluster", 
    "status": "exception", 
    "message": "Fail list: [u'10.0.1.10']", 
    "hosts": [
        {
            "status": "success", 
            "ip": "10.0.1.8", 
            "enable_sudo": true, 
            "user": "tidb", 
            "components": [
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "node_exporter", 
                    "port": "39100"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "blackbox_exporter", 
                    "port": "39115"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "prometheus", 
                    "port": "39090"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "pushgateway", 
                    "port": "39091"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "pd", 
                    "port": "32379"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "tidb", 
                    "port": "14000"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "grafana", 
                    "port": "12325"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "tikv", 
                    "port": "30160"
                }
            ], 
            "message": ""
        }, 
        {
            "status": "success", 
            "ip": "10.0.1.9", 
            "enable_sudo": true, 
            "user": "tidb", 
            "components": [
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "node_exporter", 
                    "port": "39100"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "blackbox_exporter", 
                    "port": "39115"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "tikv", 
                    "port": "30160"
                }
            ], 
            "message": ""
        }, 
        {
            "status": "success", 
            "ip": "10.0.1.11", 
            "enable_sudo": true, 
            "user": "tidb", 
            "components": [
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "node_exporter", 
                    "port": "39100"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "blackbox_exporter", 
                    "port": "39115"
                }, 
                {
                    "status": "success", 
                    "deploy_dir": "/data1/liubo/deploy", 
                    "name": "tikv", 
                    "port": "30160"
                }
            ], 
            "message": ""
        }, 
        {
            "status": "exception", 
            "ip": "10.0.1.10", 
            "message": "Failed to connect to the host via ssh", 
            "user": "tidb", 
            "components": []
        }
    ]
}
`

func (s *SyncTestSuit) TestClusterParseFile(c *C) {
	s.uuid = "f5f1ef3c-de65-439d-8d9c-b25e92b455be"
	fileName := s.uuid + ".json"
	f, err := os.Create(fileName)
	if err != nil {
		c.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(fileName)
	}()

	_, err = io.WriteString(f, topologyJsonExample)
	if err != nil {
		c.Fatal(err)
	}

	err = s.cluster.parseFile(fileName)
	if err != nil {
		c.Fatal(err)
	}

	expect := cluster{
		Name:    "test-cluster",
		Status:  "exception",
		Message: "Fail list: [u'10.0.1.10']",
		Hosts: []host{
			{
				Status:     "success",
				Ip:         "10.0.1.8",
				EnableSudo: true,
				User:       "tidb",
				Components: []component{
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "node_exporter",
						Port:      "39100",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "blackbox_exporter",
						Port:      "39115",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "prometheus",
						Port:      "39090",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "pushgateway",
						Port:      "39091",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "pd",
						Port:      "32379",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "tidb",
						Port:      "14000",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "grafana",
						Port:      "12325",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "tikv",
						Port:      "30160",
					},
				},
				Message: "",
			},
			{
				Status:     "success",
				Ip:         "10.0.1.9",
				EnableSudo: true,
				User:       "tidb",
				Components: []component{
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "node_exporter",
						Port:      "39100",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "blackbox_exporter",
						Port:      "39115",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "tikv",
						Port:      "30160",
					},
				},
				Message: "",
			},
			{
				Status:     "success",
				Ip:         "10.0.1.11",
				EnableSudo: true,
				User:       "tidb",
				Components: []component{
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "node_exporter",
						Port:      "39100",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "blackbox_exporter",
						Port:      "39115",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "tikv",
						Port:      "30160",
					},
				},
				Message: ""},
			{
				Status:     "exception",
				Ip:         "10.0.1.10",
				EnableSudo: false,
				User:       "tidb",
				Components: []component{},
				Message:    "Failed to connect to the host via ssh",
			},
		},
	}
	if !reflect.DeepEqual(s.cluster, expect) {
		c.Fatalf("want %#+v, get %#+v\n", expect, s.cluster)
	}
}

func (s *SyncTestSuit) TestClusterParserSyncTasks(c *C) {
	targetDir := "."
	tasks := s.cluster.parseSyncTasks(targetDir, s.uuid)
	expect := syncTasks{
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.11_blackbox_exporter_39115":
		syncTask{
			From:    "tidb@10.0.1.11:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.11/blackbox_exporter-39115",
			Filters: []string{"blackbox_exporter*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.11_node_exporter_39100":
		syncTask{
			From:    "tidb@10.0.1.11:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.11/node_exporter-39100",
			Filters: []string{"node_exporter*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.11_tikv_30160":
		syncTask{
			From:    "tidb@10.0.1.11:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.11/tikv-30160",
			Filters: []string{"tikv*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.8_blackbox_exporter_39115":
		syncTask{
			From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8/blackbox_exporter-39115",
			Filters: []string{"blackbox_exporter*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.8_grafana_12325":
		syncTask{
			From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8/grafana-12325",
			Filters: []string{"grafana*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.8_node_exporter_39100":
		syncTask{
			From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8/node_exporter-39100",
			Filters: []string{"node_exporter*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.8_pd_32379":
		syncTask{
			From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8/pd-32379",
			Filters: []string{"pd*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.8_prometheus_39090":
		syncTask{
			From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8/prometheus-39090",
			Filters: []string{"prometheus*", "alertmanager*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.8_pushgateway_39091":
		syncTask{
			From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8/pushgateway-39091",
			Filters: []string{"pushgateway*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.8_tidb_14000":
		syncTask{
			From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8/tidb-14000",
			Filters: []string{"tidb*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.8_tikv_30160":
		syncTask{
			From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8/tikv-30160",
			Filters: []string{"tikv*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.9_blackbox_exporter_39115":
		syncTask{
			From:    "tidb@10.0.1.9:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.9/blackbox_exporter-39115",
			Filters: []string{"blackbox_exporter*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.9_node_exporter_39100":
		syncTask{
			From:    "tidb@10.0.1.9:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.9/node_exporter-39100",
			Filters: []string{"node_exporter*"},
		},
		"f5f1ef3c-de65-439d-8d9c-b25e92b455be_10.0.1.9_tikv_30160":
		syncTask{
			From:    "tidb@10.0.1.9:/data1/liubo/deploy/log/",
			To:      "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.9/tikv-30160",
			Filters: []string{"tikv*"},
		},
	}

	if !reflect.DeepEqual(tasks, expect) {
		c.Fatalf("want %+v\n, get %#+v\n", expect, tasks)
	}
}

func (s *SyncTestSuit) TestClusterCallRsync(c *C) {
	tempDir, err := ioutil.TempDir("", "TestClusterCallRsync")
	if err != nil {
		c.Fatal(err)
	}
	deployDir := path.Join(tempDir, "deploy", "log") + "/"
	targetDir := path.Join(tempDir, "target")
	err = os.MkdirAll(deployDir, os.ModePerm)
	if err != nil {
		c.Fatal(err)
	}
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		c.Fatal(err)
	}

	fileList := []string{
		"tikv.log",
		"tikv_stderr.log",
	}
	for _, filename := range fileList {
		f, err := os.Create(path.Join(deployDir, filename))
		if err != nil {
			c.Fatal(err)
		}
		f.Close()
	}

	tasks := syncTasks{
		"test": syncTask{
			From:    deployDir,
			To:      targetDir,
			Filters: []string{"tikv*"},
		},
	}
	rsyncCfg := rsyncConfig{
		Args: []string{"-avz", fmt.Sprintf("--bwlimit=%d", 1000)},
	}
	err = callRsync(tasks, rsyncCfg)
	if err != nil {
		c.Fatal(err)
	}
	for _, filename := range fileList {
		if _, err := os.Stat(path.Join(targetDir, filename)); os.IsNotExist(err) {
			c.Fatalf("failed to rsync, file %s in target folder is not exist.\n", filename)
		}
	}
}
