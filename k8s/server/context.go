// Copyright 2021 PingCAP, Inc.
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
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/diag/api/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// diagAPICtxKey is the key to store the context
const diagAPICtxKey = "DiagAPIServerContext"

// collectJobWorker holds necessary info to manage a CollectJob
type collectJobWorker struct {
	job      *types.CollectJob
	stdout   []byte
	stderr   []byte
	cancel   chan struct{}
	checker  *checkResult
	uploader *uploadResult
}

// checkResult holds checker result of a CollectJob
type checkResult struct {
	stdout   []byte
	stderr   []byte
	cancel   chan struct{}
	finished bool
}

func (c *checkResult) reset() {
	c.stdout = make([]byte, 0)
	c.stderr = make([]byte, 0)
	c.cancel = make(chan struct{}, 1)
	c.finished = false
}

// uploadResult holds package and upload result of a CollectJob
type uploadResult struct {
	cancel chan struct{}
	date   time.Time
	status string
	result string
}

func (u *uploadResult) reset() {
	u.cancel = make(chan struct{}, 1)
	u.date = time.Now()
	u.status = ""
}

// context stores shared data of the server
type context struct {
	sync.RWMutex

	kubeCli     *kubernetes.Clientset
	dynCli      dynamic.Interface
	collectJobs map[string]*collectJobWorker
}

// newContext initializes an empty context object
func newContext() *context {
	return &context{
		collectJobs: make(map[string]*collectJobWorker),
	}
}

func (ctx *context) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(diagAPICtxKey, ctx)
		c.Next()
	}
}

// withKubeCli sets kubernetes clientset
func (ctx *context) withKubeCli(kubeCli *kubernetes.Clientset) *context {
	ctx.Lock()
	defer ctx.Unlock()

	ctx.kubeCli = kubeCli
	return ctx
}

// withDynCli sets kubernetes dynamic client
func (ctx *context) withDynCli(dynCli dynamic.Interface) *context {
	ctx.Lock()
	defer ctx.Unlock()

	ctx.dynCli = dynCli
	return ctx
}

// insertCollectJob adds a CollectJob to the list
func (ctx *context) insertCollectJob(job *types.CollectJob) *collectJobWorker {
	ctx.Lock()
	defer ctx.Unlock()

	ctx.collectJobs[job.ID] = &collectJobWorker{
		job:    job,
		stdout: make([]byte, 0),
		stderr: make([]byte, 0),
		cancel: make(chan struct{}, 1),
		checker: &checkResult{
			stdout: make([]byte, 0),
			stderr: make([]byte, 0),
			cancel: make(chan struct{}, 1),
		},
		uploader: &uploadResult{
			cancel: make(chan struct{}, 1),
		},
	}

	return ctx.collectJobs[job.ID]
}

// getCollectJob reads the CollectJob list
func (ctx *context) getCollectJobs() []*types.CollectJob {
	ctx.RLock()
	defer ctx.RUnlock()

	result := make([]*types.CollectJob, 0)
	for _, j := range ctx.collectJobs {
		job := j.job
		result = append(result, job)
	}

	return result
}

// getCollectJob reads one CollectJob from list
func (ctx *context) getCollectJob(id string) *types.CollectJob {
	if worker := ctx.getCollectWorker(id); worker != nil {
		return worker.job
	}
	return nil
}

// getCollectWorker reads one worker of CollectJob from list
func (ctx *context) getCollectWorker(id string) *collectJobWorker {
	ctx.RLock()
	defer ctx.RUnlock()

	if job, found := ctx.collectJobs[id]; found {
		return job
	}
	return nil
}

// getCollectJobOutputs get outputs of one CollectJob from list
func (ctx *context) getCollectJobOutputs(id string) ([]byte, []byte, bool) {
	ctx.RLock()
	defer ctx.RUnlock()

	if job, found := ctx.collectJobs[id]; found {
		return job.stdout, job.stderr, found
	}

	return nil, nil, false
}

// getCheckOutputs get checker outputs of one CollectJob from list
func (ctx *context) getCheckOutputs(id string) ([]byte, []byte, bool) {
	ctx.RLock()
	defer ctx.RUnlock()

	if job, found := ctx.collectJobs[id]; found {
		return job.checker.stdout, job.checker.stderr, found
	}

	return nil, nil, false
}

// getUploadOutputs get uploader outputs of one CollectJob from list
func (ctx *context) getUploadOutputs(id string) (time.Time, string, string, bool) {
	ctx.RLock()
	defer ctx.RUnlock()

	if job, found := ctx.collectJobs[id]; found {
		return job.uploader.date, job.uploader.status, job.uploader.result, found
	}

	return time.Time{}, "", "", false
}

// setJobStatus updates the status of a CollectJob, ignores if the
// job worker does not exist
func (ctx *context) setJobStatus(id, status string) {
	ctx.Lock()
	defer ctx.Unlock()

	if worker, found := ctx.collectJobs[id]; found {
		worker.job.Status = status
		return
	}
	klog.Warningf("job '%s' not found, skip setting its status to '%s'", id, status)
}
