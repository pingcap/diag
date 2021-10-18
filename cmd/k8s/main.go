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
	"strconv"

	pingcapv1alpha1 "github.com/pingcap/diag/k8s/apis/pingcap/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	klog "k8s.io/klog"
)

var _ = pingcapv1alpha1.TidbCluster{}

func init() {
	klog.InitFlags(nil)
}

func main() {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("failed to get config: %v", err)
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	tcName := os.Getenv("TC_NAME")
	if len(tcName) < 1 {
		klog.Fatal("ENV TC_NAME is not set")
	}
	tcTls := false
	tlsEnabled := os.Getenv("TC_TLS_ENABLED")
	if tlsEnabled == strconv.FormatBool(true) {
		tcTls = true
	}

	klog.Infof("initialized kube client %v", kubeCli)
	klog.Infof("initialized TLS flag as %v", tcTls)
	klog.Info("demo ended, exit.")
}
