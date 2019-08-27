package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
)

type FileInfoWithPath struct {
	os.FileInfo
	path string
}

// check if the limit exceed and truncate oldest ones
func gc(root string, interval time.Duration, threshold int64) {
	for {
		log.Info("syncer.gc: wait to start")
		time.Sleep(interval)
		log.Info("syncer.gc: start")
		if size, err := dirSize(root); err != nil {
			log.Error("syncer.gc failed to get root size:", err)
			continue
		} else if err := makeOldestFilesEmpty(root, interval, size-threshold*1024*1024*1024); err != nil {
			log.Error("syncer.gc failed to write oldest files:", err)
			continue
		}
	}
}

// make the oldest files whose size sum is more than exceed empty to release disk space
func makeOldestFilesEmpty(root string, interval time.Duration, exceed int64) error {
	if exceed <= 0 {
		// not reached threshold
		log.Info("syncer.gc: log size not exceed, skip")
		return nil
	}

	fs, err := files(root)
	if err != nil {
		return err
	}

	// sort normal files by modify time
	sort.Slice(fs, func(i, j int) bool {
		return fs[i].ModTime().Before(fs[j].ModTime())
	})

	// make the oldest empty until not exceed
	for _, f := range fs {
		if f.ModTime().Add(interval * 10).After(time.Now()) {
			// only clean the ones whose modify time is long long ago than task interval
			return fmt.Errorf("syncer.gc: logs are too new to clean, the oldest one: %s", f.path)
		}
		if f.Size() == 0 {
			continue
		}
		exceed -= f.Size()
		if err := os.Truncate(f.path, 0); err != nil {
			return err
		}
		if exceed <= 0 {
			break
		}
	}

	return nil
}

// sum up all normal files size in dir
func dirSize(dir string) (int64, error) {
	var size int64
	if fs, err := files(dir); err != nil {
		return 0, err
	} else {
		for _, f := range fs {
			size += f.Size()
		}
	}
	return size, nil
}

// return all normal files in dir
func files(dir string) ([]FileInfoWithPath, error) {
	var infos []FileInfoWithPath
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			infos = append(infos, FileInfoWithPath{info, path})
		}
		return nil
	})
	return infos, err
}
