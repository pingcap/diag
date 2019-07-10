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
                    "port": [
                        "14000", 
                        "30080"
                    ]
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
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "blackbox_exporter"},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "prometheus"},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "pushgateway"},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "pd"},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "tidb",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "grafana",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "tikv",
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
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "blackbox_exporter",
					},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "tikv",
					},
				},
				Message: "",
			},
			{
				Status:     "success",
				Ip:         "10.0.1.11",
				EnableSudo: true,
				User:       "tidb",
				Components: []component{{
					Status:    "success",
					DeployDir: "/data1/liubo/deploy",
					Name:      "node_exporter"}, {Status: "success",
					DeployDir: "/data1/liubo/deploy",
					Name:      "blackbox_exporter"},
					{
						Status:    "success",
						DeployDir: "/data1/liubo/deploy",
						Name:      "tikv",
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
		c.Fatalf("want %+v, get %+v\n", expect, s.cluster)
	}
}

func (s *SyncTestSuit) TestClusterParserSyncTasks(c *C) {
	targetDir := "."
	tasks := s.cluster.parseSyncTasks(targetDir, s.uuid)
	expect := make(syncTasks)
	expect["tidb@10.0.1.11:/data1/liubo/deploy/log/"] = "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.11"
	expect["tidb@10.0.1.8:/data1/liubo/deploy/log/"] = "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.8"
	expect["tidb@10.0.1.9:/data1/liubo/deploy/log/"] = "f5f1ef3c-de65-439d-8d9c-b25e92b455be/10.0.1.9"
	if !reflect.DeepEqual(tasks, expect) {
		c.Fatalf("want %+v, get %+v\n", expect, tasks)
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

	f, err := os.Create(path.Join(deployDir,"test.json"))
	if err != nil {
		c.Fatal(err)
	}
	defer f.Close()

	tasks := make(syncTasks)
	tasks[deployDir] = targetDir

	rsyncCfg := rsyncConfig{
		Args: []string{"-avz", fmt.Sprintf("--bwlimit=%d", 1000)},
	}
	err = callRsync(tasks, rsyncCfg)
	if err != nil {
		c.Fatal(err)
	}
	if _, err := os.Stat(path.Join(targetDir,"test.json")); os.IsNotExist(err) {
		c.Fatal("failed to rsync, file in target folder is not exist")
	}
}
