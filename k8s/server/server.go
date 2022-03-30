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
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/diag/api"
	"github.com/pingcap/diag/api/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

const apiPrefix = "api/v1"

// unified storage path
var (
	baseDir    = "/diag"
	collectDir = filepath.Join(baseDir, "collector")
	packageDir = filepath.Join(baseDir, "package")
)

// Options is the option set for diag API server
type Options struct {
	Host    string
	Port    int
	Verbose bool
}

// DiagAPIServer is the RESTful API server for diag in Kubernetes
type DiagAPIServer struct {
	engine  *gin.Engine
	address string
}

// NewServer creates a diag API server
func NewServer(opt *Options) (*DiagAPIServer, error) {
	var err error
	var cfg *rest.Config

	kubconfig := os.Getenv("KUBECONFIG")
	if kubconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get config: %v", err)
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes Clientset: %v", err)
	}
	dynCli, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes dynamic client interface: %v", err)
	}
	klog.Info("initialized kube clients")

	return &DiagAPIServer{
		engine:  newEngine(newContext().withKubeCli(kubeCli).withDynCli(dynCli), opt),
		address: fmt.Sprintf("%s:%d", opt.Host, opt.Port),
	}, nil
}

// Run starts the server
func (s *DiagAPIServer) Run() error {
	return s.engine.Run(s.address)
}

// newEngine initializes the gin engine
func newEngine(ctx *context, opt *Options) *gin.Engine {
	// set log level
	if opt.Verbose {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// create gin server
	r := gin.New()

	// add middleware here if needed
	r.Use(ginLogger())
	r.Use(ctx.middleware())
	loadJobWorker(ctx)

	// register routes
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, types.ResponseMsg{
			Message: "Page not found",
		})
	})

	// general pages

	// register apis
	apis := r.Group(apiPrefix)
	apis.GET("/", func(c *gin.Context) {
		c.FileFromFS("doc.html", http.FS(api.HTMLDocFS))
	})

	// - collectors
	apis.GET("/collectors", getJobList)
	apis.POST("/collectors", collectData)

	apis.GET("/collectors/:id", getCollectJob)
	apis.DELETE("/collectors/:id", cancelCollectJob)

	apis.GET("/collectors/:id/logs", getCollectLogs)

	apis.POST("/collectors/:id", reCollectData)

	// - data
	apis.GET("/data", getDataList)

	apis.GET("/data/:id", getDataSet)
	apis.DELETE("/data/:id", deleteDataSet)

	apis.GET("/data/:id/check", getCheckResult)
	apis.POST("/data/:id/check", checkDataSet)
	apis.DELETE("/data/:id/check", cancelCheck)

	apis.GET("/data/:id/upload", getUploadTask)
	apis.POST("/data/:id/upload", uploadDataSet)
	apis.DELETE("/data/:id/upload", cancelDataUpload)

	// - misc
	apis.GET("/version", getVersion)
	apis.GET("/status", getStatus)

	return r
}
