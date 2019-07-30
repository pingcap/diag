package main_test

import (
	"io"
	"os"
	"reflect"
	"testing"

	. "github.com/pingcap/check"
)

func TestSyncer(t *testing.T) { TestingT(t) }

type SyncTestSuit struct {
	uuid    string
	cluster *Cluster
}

var _ = Suite(&SyncTestSuit{})

var topologyJsonExample = `
{
   "cluster_name": "test-Cluster",
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
           "message": "Failed to connect to the Host via ssh",
           "user": "tidb",
           "components": []
       }
   ]
}
`
var clusterSample = &Cluster{
	Name:    "test-Cluster",
	Status:  "exception",
	Message: "Fail list: [u'10.0.1.10']",
	Hosts: []Host{
		{
			Status:     "success",
			Ip:         "10.0.1.8",
			EnableSudo: true,
			User:       "tidb",
			Components: []Component{{
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "node_exporter", Port: "39100",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "blackbox_exporter",
				Port:      "39115",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "prometheus",
				Port:      "39090",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "pushgateway", Port: "39091",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "pd", Port: "32379",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "tidb",
				Port:      "14000",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "grafana",
				Port:      "12325",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "tikv",
				Port:      "30160",
			}},
			Message: "",
		}, {
			Status:     "success",
			Ip:         "10.0.1.9",
			EnableSudo: true,
			User:       "tidb",
			Components: []Component{{
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "node_exporter",
				Port:      "39100",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "blackbox_exporter",
				Port:      "39115",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "tikv",
				Port:      "30160",
			}},
			Message: "",
		}, {
			Status:     "success",
			Ip:         "10.0.1.11",
			EnableSudo: true,
			User:       "tidb",
			Components: []Component{{
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "node_exporter",
				Port:      "39100",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "blackbox_exporter",
				Port:      "39115",
			}, {
				Status:    "success",
				DeployDir: "/data1/liubo/deploy",
				Name:      "tikv",
				Port:      "30160",
			}},
			Message: "",
		}, {
			Status:     "exception",
			Ip:         "10.0.1.10",
			EnableSudo: false,
			User:       "tidb",
			Components: []Component{},
			Message:    "Failed to connect to the Host via ssh",
		},
	},
}

func (s *SyncTestSuit) TestNewCluster(c *C) {
	s.uuid = ""
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

	cluster, err := NewCluster(fileName)
	if err != nil {
		c.Fatal(err)
	}
	expect := clusterSample

	if !reflect.DeepEqual(cluster, expect) {
		c.Fatalf("want %#+v, get %#+v\n", expect, s.cluster)
	}
}

func (s *SyncTestSuit) TestClusterLoadTasks(c *C) {
	targetDir := "."
	tasks := clusterSample.LoadTasks(targetDir, s.uuid)
	expect := []SyncTask{{
		Key:     "_10.0.1.8_node_exporter_39100",
		From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
		To:      "10.0.1.8/node_exporter-39100",
		Filters: []string{"node_exporter*"},
	}, {
		Key:     "_10.0.1.8_blackbox_exporter_39115",
		From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
		To:      "10.0.1.8/blackbox_exporter-39115",
		Filters: []string{"blackbox_exporter*"},
	}, {
		Key:     "_10.0.1.8_prometheus_39090",
		From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
		To:      "10.0.1.8/prometheus-39090",
		Filters: []string{"prometheus*", "alertmanager*"},
	}, {
		Key:     "_10.0.1.8_pushgateway_39091",
		From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
		To:      "10.0.1.8/pushgateway-39091",
		Filters: []string{"pushgateway*"},
	}, {
		Key:     "_10.0.1.8_pd_32379",
		From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
		To:      "10.0.1.8/pd-32379",
		Filters: []string{"pd*"},
	}, {
		Key:     "_10.0.1.8_tidb_14000",
		From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
		To:      "10.0.1.8/tidb-14000",
		Filters: []string{"tidb*"},
	}, {
		Key:     "_10.0.1.8_grafana_12325",
		From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
		To:      "10.0.1.8/grafana-12325",
		Filters: []string{"grafana*"},
	}, {
		Key:     "_10.0.1.8_tikv_30160",
		From:    "tidb@10.0.1.8:/data1/liubo/deploy/log/",
		To:      "10.0.1.8/tikv-30160",
		Filters: []string{"tikv*"},
	}, {
		Key:     "_10.0.1.9_node_exporter_39100",
		From:    "tidb@10.0.1.9:/data1/liubo/deploy/log/",
		To:      "10.0.1.9/node_exporter-39100",
		Filters: []string{"node_exporter*"},
	}, {
		Key:     "_10.0.1.9_blackbox_exporter_39115",
		From:    "tidb@10.0.1.9:/data1/liubo/deploy/log/",
		To:      "10.0.1.9/blackbox_exporter-39115",
		Filters: []string{"blackbox_exporter*"},
	}, {
		Key:     "_10.0.1.9_tikv_30160",
		From:    "tidb@10.0.1.9:/data1/liubo/deploy/log/",
		To:      "10.0.1.9/tikv-30160",
		Filters: []string{"tikv*"},
	}, {
		Key:     "_10.0.1.11_node_exporter_39100",
		From:    "tidb@10.0.1.11:/data1/liubo/deploy/log/",
		To:      "10.0.1.11/node_exporter-39100",
		Filters: []string{"node_exporter*"},
	}, {
		Key:     "_10.0.1.11_blackbox_exporter_39115",
		From:    "tidb@10.0.1.11:/data1/liubo/deploy/log/",
		To:      "10.0.1.11/blackbox_exporter-39115",
		Filters: []string{"blackbox_exporter*"},
	}, {
		Key:     "_10.0.1.11_tikv_30160",
		From:    "tidb@10.0.1.11:/data1/liubo/deploy/log/",
		To:      "10.0.1.11/tikv-30160",
		Filters: []string{"tikv*"},
	}}

	if !reflect.DeepEqual(tasks, expect) {
		c.Fatalf("want %#+v\n, get %#+v\n", expect, tasks)
	}
}
