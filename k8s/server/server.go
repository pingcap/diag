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

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const apiPrefix = "api/v1"

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
	cfg, err := rest.InClusterConfig()
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

	// register routes
	apis := r.Group(apiPrefix)
	// register apis
	// - collectors
	apis.GET("/collectors", getJobList)
	apis.POST("/collectors", collectData)
	apis.GET("/collectors/:id", getCollectJob)

	// - data
	apis.GET("/data", getDataList)
	apis.GET("/data/:id", getDataSet)
	apis.DELETE("/data/:id", deleteDataSet)

	// - misc
	apis.GET("/version", getVersion)

	return r
}
