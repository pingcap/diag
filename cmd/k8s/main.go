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

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingcap/diag/collector"
	"github.com/pingcap/diag/version"
	"github.com/pingcap/tiup/pkg/set"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	klog "k8s.io/klog"
)

var (
	cm   *collector.Manager
	cOpt collector.CollectOptions
	opt  collector.BaseOptions
)

func init() {
	klog.InitFlags(nil)
	cm = collector.NewEmptyManager("tidb")
	cOpt = collector.CollectOptions{
		Include: set.NewStringSet( // collect all types by default
			collector.CollectTypeSystem,
			collector.CollectTypeMonitor,
			collector.CollectTypeLog,
			collector.CollectTypeConfig,
		),
		Exclude: set.NewStringSet(),
	}
	opt = collector.BaseOptions{}
}

func main() {
	klog.Infof("started diag pod %s", version.String())

	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("failed to get config: %v", err)
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}
	dynCli, err := dynamic.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes dynamic client interface: %v", err)
	}
	klog.Info("initialized kube clients")

	// run collectors
	cOpt.Mode = collector.CollectModeK8s
	opt.Cluster = "m31"
	opt.ScrapeBegin = time.Now().Add(time.Hour * -2).Format(time.RFC3339)
	opt.ScrapeEnd = time.Now().Format(time.RFC3339)
	if err := cm.CollectClusterInfo(&opt, &cOpt, nil, kubeCli, dynCli); err != nil {
		klog.Errorf("error collecting info: %s", err)
	}

	klog.Info("demo ended, sleep forever.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	sig := <-sc
	klog.Infof("got signal %s, exit", sig)
}
