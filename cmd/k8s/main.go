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
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	jsoniter "github.com/json-iterator/go"
	pingcapv1alpha1 "github.com/pingcap/diag/k8s/apis/pingcap/v1alpha1"
	"github.com/pingcap/diag/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	klog "k8s.io/klog"
)

func init() {
	klog.InitFlags(nil)
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
	gvr := schema.GroupVersionResource{
		Group:    "pingcap.com",
		Version:  "v1alpha1",
		Resource: "tidbclusters",
	}

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		klog.Fatal("NAMESPACE environment variable not set")
	}
	tcName := os.Getenv("TC_NAME")
	if len(tcName) < 1 {
		klog.Fatal("ENV TC_NAME is not set")
	}

	klog.Info("initialized kube clients")

	podList, err := kubeCli.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("failed to list pods in namespace %s: %v", ns, err)
	}
	klog.Infof("listed pods in namespace %s:", ns)
	for _, pod := range podList.Items {
		podName := pod.Name
		cTime := pod.CreationTimestamp
		hostIP := pod.Status.HostIP
		podIPs := pod.Status.PodIPs
		podStatus := pod.Status.Phase
		klog.Infof("%s (%s) on %s, %s, created at %s", podName, podIPs[0], hostIP, podStatus, cTime)
	}

	svcList, err := kubeCli.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("failed to list services in namespace %s: %v", ns, err)
	}
	klog.Infof("listed services in namespace %s:", ns)
	for _, svc := range svcList.Items {
		svcName := svc.Name
		svcType := svc.Spec.Type
		var svcIP string
		var svcPort string
		switch svcType {
		case corev1.ServiceTypeClusterIP:
			svcIP = svc.Spec.ClusterIP
			ports := make([]string, 0)
			for _, p := range svc.Spec.Ports {
				svcPort := p.Port
				svcTarget := p.TargetPort
				portName := p.Name
				portProto := p.Protocol
				ports = append(ports,
					fmt.Sprintf("%d->%s(%s:%s)", svcPort, svcTarget.String(), portProto, portName),
				)
			}
			svcPort = strings.Join(ports, ",")
		case corev1.ServiceTypeNodePort:
			svcIP = "*"
			ports := make([]string, 0)
			for _, p := range svc.Spec.Ports {
				svcPort := p.NodePort
				svcTarget := p.TargetPort
				portName := p.Name
				portProto := p.Protocol
				ports = append(ports,
					fmt.Sprintf("%d->%s(%s:%s)", svcPort, svcTarget.String(), portProto, portName),
				)
			}
			svcPort = strings.Join(ports, ",")
		case corev1.ServiceTypeLoadBalancer:
			svcIP = svc.Spec.LoadBalancerIP
			ports := make([]string, 0)
			for _, p := range svc.Spec.Ports {
				svcPort := p.Port
				svcTarget := p.TargetPort
				portName := p.Name
				portProto := p.Protocol
				ports = append(ports,
					fmt.Sprintf("%d->%s(%s:%s)", svcPort, svcTarget.String(), portProto, portName),
				)
			}
			svcPort = strings.Join(ports, ",")
		}

		klog.Infof("%s (%s) %s %s", svcName, svcType, svcIP, svcPort)
	}

	tcList, err := dynCli.Resource(gvr).Namespace(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("failed to list tidbclusters in namespace %s: %v", ns, err)
	}
	tcData, err := tcList.MarshalJSON()
	if err != nil {
		klog.Fatalf("failed to marshal tidbclusters to json: %v", err)
	}
	var tcs pingcapv1alpha1.TidbClusterList
	if err := jsoniter.Unmarshal(tcData, &tcs); err != nil {
		klog.Fatalf("failed to unmarshal tidbclusters crd: %v", err)
	}
	klog.Infof("listed %d tidbclusters:", len(tcs.Items))
	for _, tc := range tcs.Items {
		clsName := tc.ObjectMeta.Name
		cTime := tc.ObjectMeta.CreationTimestamp
		status := tc.Status.Conditions[0].Type
		klog.Infof("TiDB Cluster '%s': %s, %s, created at %s",
			clsName, tc.Spec.Version, status, cTime)
		if tc.Spec.PD != nil {
			klog.Infof("  PD:      %d  %s (%s)", tc.Spec.PD.Replicas, tc.Status.PD.Phase, tc.Status.PD.Image)
			for _, member := range tc.Status.PD.Members {
				var status string
				if member.Health {
					status = "healthy"
				} else {
					status = "unhealthy"
				}
				klog.Infof("    %s, %s, %s", member.Name, status, member.ClientURL)
			}
		}
		if tc.Spec.TiDB != nil {
			klog.Infof("  TiDB:    %d  %s (%s)", tc.Spec.TiDB.Replicas, tc.Status.TiDB.Phase, tc.Status.TiDB.Image)
		}
		if tc.Spec.TiKV != nil {
			klog.Infof("  TiKV:    %d  %s (%s)", tc.Spec.TiKV.Replicas, tc.Status.TiKV.Phase, tc.Status.TiKV.Image)
		}
		if tc.Spec.TiFlash != nil {
			klog.Infof("  TiFlash: %d  %s (%s)", tc.Spec.TiFlash.Replicas, tc.Status.TiFlash.Phase, tc.Status.TiFlash.Image)
		}
		if tc.Spec.TiCDC != nil {
			klog.Infof("  TiCDC:   %d  %s", tc.Spec.TiCDC.Replicas, tc.Status.TiCDC.Phase)
		}
		if tc.Spec.Pump != nil {
			klog.Infof("  Pump:    %d  %s", tc.Spec.Pump.Replicas, tc.Status.Pump.Phase)
		}
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
