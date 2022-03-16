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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/k8s/apis/label"
	pingcapv1alpha1 "github.com/pingcap/diag/k8s/apis/pingcap/v1alpha1"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// prepareArgsForK8sCluster parses arguments and create output dir for tiup-operator
// deployed tidb clusters
func (m *Manager) prepareArgsForK8sCluster(
	opt *BaseOptions,
	cOpt *CollectOptions,
) (string, error) {
	// parse time range
	end, err := utils.ParseTime(opt.ScrapeEnd)
	if err != nil {
		return "", err
	}
	// if the begin time point is a minus integer, assume it as hour offset
	var start time.Time
	if offset, err := strconv.Atoi(opt.ScrapeBegin); err == nil && offset < 0 {
		start = end.Add(time.Hour * time.Duration(offset))
	} else {
		start, err = utils.ParseTime(opt.ScrapeBegin)
		if err != nil {
			return "", err
		}
	}

	if start.After(end) {
		return "", fmt.Errorf("end time cannot be earlier than start time")
	}

	// update time strings in setting to ensure all collectors work properly
	opt.ScrapeBegin = start.Format(time.RFC3339)
	opt.ScrapeEnd = end.Format(time.RFC3339)

	return m.getOutputDir(cOpt.Dir)
}

// buildTopoForK8sCluster creates an abstract topo from tiup-cluster metadata
func buildTopoForK8sCluster(
	_ *Manager,
	opt *BaseOptions,
	kubeCli *kubernetes.Clientset,
	dynCli dynamic.Interface,
) (
	*models.TiDBCluster,
	*pingcapv1alpha1.TidbCluster,
	*pingcapv1alpha1.TidbMonitor,
	error,
) {
	gvrTiDB := schema.GroupVersionResource{
		Group:    "pingcap.com",
		Version:  "v1alpha1",
		Resource: "tidbclusters",
	}
	gvrMonitor := schema.GroupVersionResource{
		Group:    "pingcap.com",
		Version:  "v1alpha1",
		Resource: "tidbmonitors",
	}

	// get namespace
	ns := opt.Namespace
	if opt.Namespace == "" {
		ns = os.Getenv("NAMESPACE")
		if ns == "" {
			msg := "namespace not specified and NAMESPACE environment variable not set"
			klog.Error(msg)
			return nil, nil, nil, fmt.Errorf(msg)
		}
		klog.Infof("got namespace '%s'", ns)
	}

	tcList, err := dynCli.Resource(gvrTiDB).Namespace(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		msg := fmt.Sprintf("failed to list tidbclusters in namespace %s: %v", ns, err)
		klog.Errorf(msg)
		return nil, nil, nil, fmt.Errorf(msg)
	}
	tcData, err := tcList.MarshalJSON()
	if err != nil {
		msg := fmt.Sprintf("failed to marshal tidbclusters to json: %v", err)
		klog.Errorf(msg)
		return nil, nil, nil, fmt.Errorf(msg)
	}
	var tcs pingcapv1alpha1.TidbClusterList
	if err := json.Unmarshal(tcData, &tcs); err != nil {
		msg := fmt.Sprintf("failed to unmarshal tidbclusters crd: %v", err)
		klog.Errorf(msg)
		return nil, nil, nil, fmt.Errorf(msg)
	}

	klog.Infof("found %d tidbclusters in namespace '%s'", len(tcs.Items), ns)

	monList, err := dynCli.Resource(gvrMonitor).Namespace(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		msg := fmt.Sprintf("failed to list tidbmonitors in namespace %s: %v", ns, err)
		klog.Errorf(msg)
		return nil, nil, nil, fmt.Errorf(msg)
	}
	monData, err := monList.MarshalJSON()
	if err != nil {
		msg := fmt.Sprintf("failed to marshal tidbmonitors to json: %v", err)
		klog.Errorf(msg)
		return nil, nil, nil, fmt.Errorf(msg)
	}
	var mon pingcapv1alpha1.TidbMonitorList
	if err := json.Unmarshal(monData, &mon); err != nil {
		msg := fmt.Sprintf("failed to unmarshal tidbmonitors crd: %v", err)
		klog.Errorf(msg)
		return nil, nil, nil, fmt.Errorf(msg)
	}

	klog.Infof("found %d tidbmonitors in namespace '%s'", len(mon.Items), ns)

	cls := &models.TiDBCluster{Namespace: ns}
	var cluster *pingcapv1alpha1.TidbCluster

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
		cluster = &tc //#nosec G601
		cls.Version = tc.Spec.Version

		for _, ins := range tc.Status.PD.Members {
			if len(cls.PD) < 1 {
				cls.PD = make([]*models.PDSpec, 0)
			}

			pod, err := kubeCli.CoreV1().Pods(ns).Get(context.TODO(), ins.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("error getting pod '%s' in '%s': %v", ins.Name, ns, err)
			}
			if pod.Status.Phase != corev1.PodRunning {
				klog.Warningf("pod '%s' is in '%s' status, skip it", ins.Name, pod.Status.Phase)
				continue
			}

			cls.PD = append(cls.PD, &models.PDSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       pod.Status.PodIP,
					Port:       2379,
					StatusPort: 2379,
					Attributes: map[string]interface{}{
						"health":     ins.Health,
						"id":         ins.ID,
						"client_url": ins.ClientURL,
						"image":      tc.Status.PD.Image,
						"pod":        ins.Name,
						"domain":     fmt.Sprintf("%s.%s-pd-peer.%s.svc", ins.Name, clsName, ns),
					},
				},
			})
		}
		for _, ins := range tc.Status.TiDB.Members {
			if tc.Spec.TiDB != nil {
				if len(cls.TiDB) < 1 {
					cls.TiDB = make([]*models.TiDBSpec, 0)
				}

				pod, err := kubeCli.CoreV1().Pods(ns).Get(context.TODO(), ins.Name, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("error getting pod '%s' in '%s': %v", ins.Name, ns, err)
				}
				if pod.Status.Phase != corev1.PodRunning {
					klog.Warningf("pod '%s' is in '%s' status, skip it", ins.Name, pod.Status.Phase)
					continue
				}

				cls.TiDB = append(cls.TiDB, &models.TiDBSpec{
					ComponentSpec: models.ComponentSpec{
						Host:       pod.Status.PodIP,
						Port:       4000,
						StatusPort: 10080,
						Attributes: map[string]interface{}{
							"health": ins.Health,
							"image":  tc.Status.TiDB.Image,
							"pod":    ins.Name,
							"domain": fmt.Sprintf("%s.%s-tidb-peer.%s.svc", ins.Name, clsName, ns),
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

				pod, err := kubeCli.CoreV1().Pods(ns).Get(context.TODO(), ins.PodName, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("error getting pod '%s' in '%s': %v", ins.PodName, ns, err)
				}
				if pod.Status.Phase != corev1.PodRunning {
					klog.Warningf("pod '%s' is in '%s' status, skip it", ins.PodName, pod.Status.Phase)
					continue
				}

				cls.TiKV = append(cls.TiKV, &models.TiKVSpec{
					ComponentSpec: models.ComponentSpec{
						Host:       pod.Status.PodIP,
						Port:       20160,
						StatusPort: 20180,
						Attributes: map[string]interface{}{
							"state":        ins.State,
							"id":           ins.ID,
							"leader_count": ins.LeaderCount,
							"image":        tc.Status.TiKV.Image,
							"pod":          ins.PodName,
							"domain":       fmt.Sprintf("%s.%s-tikv-peer.%s.svc", ins.PodName, clsName, ns),
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

				pod, err := kubeCli.CoreV1().Pods(ns).Get(context.TODO(), ins.PodName, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("error getting pod '%s' in '%s': %v", ins.PodName, ns, err)
				}
				if pod.Status.Phase != corev1.PodRunning {
					klog.Warningf("pod '%s' is in '%s' status, skip it", ins.PodName, pod.Status.Phase)
					continue
				}

				cls.TiFlash = append(cls.TiFlash, &models.TiFlashSpec{
					ComponentSpec: models.ComponentSpec{
						Host:       pod.Status.PodIP,
						Port:       3930,
						StatusPort: 20292,
						Attributes: map[string]interface{}{
							"state":        ins.State,
							"id":           ins.ID,
							"leader_count": ins.LeaderCount,
							"image":        tc.Status.TiFlash.Image,
							"pod":          ins.PodName,
							"domain":       fmt.Sprintf("%s.%s-tiflash-peer.%s.svc", ins.PodName, clsName, ns),
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

				pod, err := kubeCli.CoreV1().Pods(ns).Get(context.TODO(), ins.PodName, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("error getting pod '%s' in '%s': %v", ins.PodName, ns, err)
				}
				if pod.Status.Phase != corev1.PodRunning {
					klog.Warningf("pod '%s' is in '%s' status, skip it", ins.PodName, pod.Status.Phase)
					continue
				}

				cls.TiCDC = append(cls.TiCDC, &models.TiCDCSpec{
					ComponentSpec: models.ComponentSpec{
						Host: pod.Status.PodIP,
						Port: 8301,
						Attributes: map[string]interface{}{
							"id":     ins.ID,
							"pod":    ins.PodName,
							"domain": fmt.Sprintf("%s.%s-ticdc-peer.%s.svc", ins.PodName, clsName, ns),
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

				pod, err := kubeCli.CoreV1().Pods(ns).Get(context.TODO(), ins.Host, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("error getting pod '%s' in '%s': %v", ins.Host, ns, err)
				}
				if pod.Status.Phase != corev1.PodRunning {
					klog.Warningf("pod '%s' is in '%s' status, skip it", ins.Host, pod.Status.Phase)
					continue
				}

				cls.Pump = append(cls.Pump, &models.PumpSpec{
					ComponentSpec: models.ComponentSpec{
						Host: pod.Status.PodIP,
						Port: 8250,
						Attributes: map[string]interface{}{
							"node":  ins.NodeID,
							"state": ins.State,
							"pod":   ins.Host,
						},
					},
				})
			}
		}
	}

	// find monitor pod
	var matchedMon pingcapv1alpha1.TidbMonitor
	for i, m := range mon.Items {
		monName := m.ObjectMeta.Name
		matched := false
		for _, clsRef := range m.Spec.Clusters {
			if clsRef.Name == opt.Cluster {
				matched = true
				break
			}
		}
		if !matched {
			klog.Infof("monitor %d ('%s') is not the one we want to collect ('%s'), skip.", i, monName, opt.Cluster)
			continue
		}

		cTime := m.ObjectMeta.CreationTimestamp
		klog.Infof("found monitor '%s', created at %s", monName, cTime)
		matchedMon = m
		break
	}

	labels := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			label.ManagedByLabelKey: "tidb-operator",
			label.NameLabelKey:      "tidb-cluster",
			label.ComponentLabelKey: "monitor",
			label.InstanceLabelKey:  matchedMon.Name,
			label.UsedByLabelKey:    "prometheus",
		},
	}
	selector, err := metav1.LabelSelectorAsSelector(labels)
	if err != nil {
		klog.Error(err)
		return nil, nil, nil, err
	}
	svcs, err := kubeCli.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		klog.Errorf("error listing services of '%s' in '%s': %v", matchedMon.Name, ns, err)
	}

	klog.Infof("found %d services in '%s/%s'", len(svcs.Items), ns, matchedMon.Name)

	for _, svc := range svcs.Items {
		// svc.Spec.ClusterIPs is not available on k8s v1.19.x
		if svc.Spec.ClusterIP == "" {
			klog.Errorf("service %s does not have any clusterIP, skip", svc.Name)
			continue
		}
		ip := svc.Spec.ClusterIP
		port := 0

		for _, p := range svc.Spec.Ports {
			if p.Name == "http-prometheus" {
				port = int(p.Port)
			}
			break
		}
		if port == 0 {
			continue
		}

		if len(cls.Monitors) < 1 {
			cls.Monitors = make([]*models.MonitorSpec, 0)
		}
		cls.Monitors = append(cls.Monitors, &models.MonitorSpec{
			ComponentSpec: models.ComponentSpec{
				Host: ip,
				Port: port,
				Attributes: map[string]interface{}{
					"service": svc.Name,
				},
			},
		})
	}

	return cls, cluster, &matchedMon, nil
}

// GetClusterInfoFromFile
func GetClusterInfoFromFile(path string) (*ClusterJSON, error) {
	c := &ClusterJSON{}

	fbytes, err := ioutil.ReadFile(filepath.Join(path, FileNameClusterJSON))
	if err != nil {
		return c, err
	}

	err = json.Unmarshal(fbytes, c)
	if err != nil {
		return c, err
	}

	return c, nil
}
