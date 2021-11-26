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
	goctx "context"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/diag/api/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// diagAPICtxKey is the key to store the context
const diagAPICtxKey = "DiagAPIServerContext"

// collectJobCtx holds necessary info to manage a CollectJob
type collectJobCtx struct {
	job    *types.CollectJob
	cancel chan struct{}
}

// context stores shared data of the server
type context struct {
	sync.RWMutex
	ctx goctx.Context

	kubeCli     *kubernetes.Clientset
	dynCli      dynamic.Interface
	collectJobs map[string]*collectJobCtx
}

// newContext initializes an empty context object
func newContext() *context {
	return &context{
		collectJobs: make(map[string]*collectJobCtx),
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
func (ctx *context) insertCollectJob(job *types.CollectJob) *collectJobCtx {
	ctx.Lock()
	defer ctx.Unlock()

	return &collectJobCtx{
		job:    job,
		cancel: make(chan struct{}, 1),
	}
}

// getCollectJob reads the CollectJob list
func (ctx *context) getCollectJob() []*types.CollectJob {
	ctx.RLock()
	defer ctx.RUnlock()

	result := make([]*types.CollectJob, 0)
	for _, j := range ctx.collectJobs {
		result = append(result, j.job)
	}

	return result
}
