package main

import (
	"fmt"
	"os"
	"path"
	"time"

	. "github.com/pingcap/check"
)

type taskManagerTestSuit struct{}

var _ = Suite(&taskManagerTestSuit{})

func (s *taskManagerTestSuit) TestRunTasks(c *C) {
	from, to := createTestTempDir("TestRunTasks", c)
	createLogFiles(from, c)

	manager := TaskManager{
		Interval: 1 * time.Second,
		Cfg: RsyncConfig{
			Args: []string{"-avz", fmt.Sprintf("--bwlimit=%d", 1000)},
		},
		StopCh: make(chan struct{}),
		cancel: func() {},
	}
	task := SyncTask{
		Key:     "tikv_1",
		From:    from,
		To:      to,
		Filters: []string{"tikv*"},
	}
	manager.RunTasks([]SyncTask{task})
	go func() {
		time.Sleep(2 * time.Second)
		task := SyncTask{
			Key:     "tikv_2",
			From:    from,
			To:      to,
			Filters: []string{"tikv*"},
		}
		manager.RunTasks([]SyncTask{task})
	}()
	go func() {
		time.Sleep(5 * time.Second)
		manager.RunTasks([]SyncTask{})
	}()
	go func() {
		time.Sleep(7 * time.Second)
		manager.Stop()
	}()
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
	tempDir, err := os.MkdirTemp("", name)
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
