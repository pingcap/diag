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

package collector

import (
	"context"
	"os"

	jsoniter "github.com/json-iterator/go"
	pingcapv1alpha1 "github.com/pingcap/diag/k8s/apis/pingcap/v1alpha1"
	"github.com/pingcap/diag/pkg/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// buildTopoForK8sCluster creates an abstract topo from tiup-cluster metadata
func buildTopoForK8sCluster(
	_ *Manager,
	opt *BaseOptions,
	_ *kubernetes.Clientset,
	dynCli dynamic.Interface,
) (*models.TiDBCluster, error) {
	gvr := schema.GroupVersionResource{
		Group:    "pingcap.com",
		Version:  "v1alpha1",
		Resource: "tidbclusters",
	}

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		klog.Fatal("NAMESPACE environment variable not set")
	}
	klog.Infof("got namespace '%s'", ns)

	/*
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
	*/

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

	klog.Infof("found %d tidbclusters in namespace '%s'", len(tcs.Items), ns)
	cls := &models.TiDBCluster{}

	for i, tc := range tcs.Items {
		clsName := tc.ObjectMeta.Name
		if clsName != opt.Cluster {
			klog.Infof("cluster %d ('%s') is not the one we want to collect ('%s'), skip.", i, clsName, opt.Cluster)
			continue
		}

		cTime := tc.ObjectMeta.CreationTimestamp
		status := tc.Status.Conditions[0].Type
		klog.Infof("found cluster '%s': %s, %s, created at %s",
			clsName, tc.Spec.Version, status, cTime)

		for _, ins := range tc.Status.PD.Members {
			if len(cls.PD) < 1 {
				cls.PD = make([]*models.PDSpec, 0)
			}
			cls.PD = append(cls.PD, &models.PDSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       ins.Name,
					Port:       2379,
					StatusPort: 2379,
					Attributes: map[string]interface{}{
						"health":     ins.Health,
						"id":         ins.ID,
						"client_url": ins.ClientURL,
						"image":      tc.Status.PD.Image,
					},
				},
			})
		}
		for _, ins := range tc.Status.TiDB.Members {
			if tc.Spec.TiDB != nil {
				if len(cls.TiDB) < 1 {
					cls.TiDB = make([]*models.TiDBSpec, 0)
				}
				cls.TiDB = append(cls.TiDB, &models.TiDBSpec{
					ComponentSpec: models.ComponentSpec{
						Host:       ins.Name,
						Port:       4000,
						StatusPort: 10080,
						Attributes: map[string]interface{}{
							"health": ins.Health,
							"image":  tc.Status.TiDB.Image,
						},
					},
				})
			}
		}
		for _, ins := range tc.Status.TiKV.Stores {
			if tc.Spec.TiKV != nil {
				if len(cls.TiKV) < 1 {
					cls.TiKV = make([]*models.TiKVSpec, 0)
				}
				cls.TiKV = append(cls.TiKV, &models.TiKVSpec{
					ComponentSpec: models.ComponentSpec{
						Host:       ins.PodName,
						Port:       20160,
						StatusPort: 20180,
						Attributes: map[string]interface{}{
							"state":        ins.State,
							"id":           ins.ID,
							"leader_count": ins.LeaderCount,
							"image":        tc.Status.TiKV.Image,
						},
					},
				})
			}
		}
		for _, ins := range tc.Status.TiFlash.Stores {
			if tc.Spec.TiFlash != nil {
				if len(cls.TiFlash) < 1 {
					cls.TiFlash = make([]*models.TiFlashSpec, 0)
				}
				cls.TiFlash = append(cls.TiFlash, &models.TiFlashSpec{
					ComponentSpec: models.ComponentSpec{
						Host:       ins.PodName,
						Port:       3930,
						StatusPort: 20292,
						Attributes: map[string]interface{}{
							"state":        ins.State,
							"id":           ins.ID,
							"leader_count": ins.LeaderCount,
							"image":        tc.Status.TiFlash.Image,
						},
					},
				})
			}
		}
		for _, ins := range tc.Status.TiCDC.Captures {
			if tc.Spec.TiCDC != nil {
				if len(cls.TiCDC) < 1 {
					cls.TiCDC = make([]*models.TiCDCSpec, 0)
				}
				cls.TiCDC = append(cls.TiCDC, &models.TiCDCSpec{
					ComponentSpec: models.ComponentSpec{
						Host: ins.PodName,
						Port: 8301,
						Attributes: map[string]interface{}{
							"id": ins.ID,
						},
					},
				})
			}
		}
		for _, ins := range tc.Status.Pump.Members {
			if tc.Spec.Pump != nil {
				if len(cls.Pump) < 1 {
					cls.Pump = make([]*models.PumpSpec, 0)
				}
				cls.Pump = append(cls.Pump, &models.PumpSpec{
					ComponentSpec: models.ComponentSpec{
						Host: ins.Host,
						Port: 8250,
						Attributes: map[string]interface{}{
							"node":  ins.NodeID,
							"state": ins.State,
						},
					},
				})
			}
		}
	}

	return cls, nil
}
