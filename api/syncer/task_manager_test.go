package syncer_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	. "github.com/pingcap/check"
	"github.com/pingcap/tidb-foresight/api/syncer"
)

type taskManagerTestSuit struct{}

var _ = Suite(&taskManagerTestSuit{})

func (s *taskManagerTestSuit) TestRunTasks(c *C) {
	from, to := createTestTempDir("TestRunTasks", c)
	createLogFiles(from, c)

	manager := syncer.TaskManager{
		Interval: 1 * time.Second,
		Cfg: syncer.RsyncConfig{
			Args: []string{"-avz", fmt.Sprintf("--bwlimit=%d", 1000)},
		},
		TodoTaskCh: make(chan syncer.SyncTask, 1),
		StopCh: make(chan struct{}),
	}
	task := syncer.SyncTask{
		Key:     "tikv_1",
		From:    from,
		To:      to,
		Filters: []string{"tikv*"},
	}
	manager.RunTasks([]syncer.SyncTask{task})
	go func() {
		time.Sleep(2 * time.Second)
		task := syncer.SyncTask{
			Key:     "tikv_2",
			From:    from,
			To:      to,
			Filters: []string{"tikv*"},
		}
		manager.RunTasks([]syncer.SyncTask{task})
	}()
	go func() {
		time.Sleep(5 * time.Second)
		manager.RunTasks([]syncer.SyncTask{})
	}()
	go func() {
		time.Sleep(7 * time.Second)
		manager.Stop()
	}()
	manager.Start()
}

func createLogFiles(deployDir string, c *C) {
	fileList := []string{
		"tikv.log",
		"tikv_stderr.log",
	}
	for _, filename := range fileList {
		f, err := os.Create(path.Join(deployDir, filename))
		if err != nil {
			c.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			c.Fatal(err)
		}
	}
}

func createTestTempDir(name string, c *C) (string, string) {
	tempDir, err := ioutil.TempDir("", name)
	if err != nil {
		c.Fatal(err)
	}
	from := path.Join(tempDir, "deploy", "log") + "/"
	to := path.Join(tempDir, "target")
	err = os.MkdirAll(from, os.ModePerm)
	if err != nil {
		c.Fatal(err)
	}
	err = os.MkdirAll(to, os.ModePerm)
	if err != nil {
		c.Fatal(err)
	}
	return from, to
}
