// Copyright 2022 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pingcap/diag/api/types"
	"github.com/pingcap/diag/collector"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
	"k8s.io/klog/v2"
)

func loadJobWorker(ctx *context) {

	if tiuputils.IsNotExist(collectDir) {
		os.MkdirAll(collectDir, 0755)
	}

	// get file list
	fileList, err := filepath.Glob(filepath.Join(collectDir, "*"))
	if err != nil {
		klog.Warningf("load worker from %s failed: %s", collectDir, err)
	}

	klog.Infof("get diag collect file list %v", fileList)

	for _, f := range fileList {
		fname := filepath.Base(f)
		if !strings.HasPrefix(fname, "diag-") {
			klog.Warningf("unknown diag collect directory %s from %s", fname, collectDir)
			continue
		}
		// load worker
		fi, err := os.Stat(f)
		if err != nil {
			klog.Warningf("load worker from %s failed: %v", f, err)
			continue
		}
		if !fi.IsDir() {
			klog.Warningf("%s is not directory from %s", fname, collectDir)
			continue
		}

		id := strings.Split(fname, "-")[1]
		c, err := collector.GetClusterInfoFromFile(f)

		status := taskStatusFinish
		if err != nil {
			klog.Warningf("load worker [%s] failed: %v", id, err)
			status = taskStatusError
		}
		job := &types.CollectJob{
			ID:          strings.Split(f, "-")[1],
			Status:      status,
			ClusterName: c.ClusterName,
			Collectors:  c.Collectors,
			From:        c.BeginTime,
			To:          c.EndTime,
			Date:        time.Now().Format(time.RFC3339),
			Dir:         f,
		}
		ctx.insertCollectJob(job)
		klog.Infof("load worker [%s] success", id)

	}

	klog.Infof("finished loading workers from %s ", collectDir)
}
