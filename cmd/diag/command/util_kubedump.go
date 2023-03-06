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

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pingcap/diag/collector"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

func newkubeDumpCmd() *cobra.Command {
	opt := collector.BaseOptions{}
	cOpt := collector.CollectOptions{}
	cOpt.Collectors, _ = collector.ParseCollectTree([]string{collector.CollectTypeMonitor}, nil)
	var (
		clsName  string
		clsID    string
		caPath   string
		certPath string
		keyPath  string
	)

	cmd := &cobra.Command{
		Use:   "kubedump",
		Short: "Dump TSDB files from a Prometheus pod.",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.SetDisplayModeFromString(gOpt.DisplayMode)

			cfg, err := clientcmd.BuildConfigFromFlags("", opt.Kubeconfig)
			if err != nil {
				return err
			}
			clsID, cOpt.PodName, err = bbuildTopoForK8sCluster(clsName, opt.Namespace, cfg)
			if err != nil {
				return err
			}

			cm := collector.NewManager("tidb", nil, log)

			cOpt.Collectors, _ = collector.ParseCollectTree([]string{"monitor.metric"}, nil)
			opt.Cluster = clsName
			cOpt.RawMonitor = true
			cOpt.RawRequest = strings.Join(os.Args[1:], " ")
			cOpt.Mode = collector.CollectModeManual      // set collect mode
			cOpt.ExtendedAttrs = make(map[string]string) // init attributes map
			cOpt.ExtendedAttrs[collector.AttrKeyClusterID] = clsID
			cOpt.ExtendedAttrs[collector.AttrKeyTLSCAFile] = caPath
			cOpt.ExtendedAttrs[collector.AttrKeyTLSCertFile] = certPath
			cOpt.ExtendedAttrs[collector.AttrKeyTLSKeyFile] = keyPath

			_, err = cm.CollectClusterInfo(&opt, &cOpt, &gOpt, nil, nil, skipConfirm)

			return err
		},
	}

	cmd.Flags().StringVar(&opt.Kubeconfig, "kubeconfig", "", "path of kubeconfig")
	cmd.Flags().StringVar(&clsName, "name", "", "name of the TiDB cluster")
	//cmd.Flags().StringVar(&clsID, "cluster-id", "", "ID of the TiDB cluster")
	cmd.Flags().StringVar(&opt.Namespace, "namespace", "", "namespace of prometheus")
	//cmd.Flags().StringVar(&cOpt.PodName, "pod", "", "pod name of prometheus")
	//cmd.Flags().StringVar(&cOpt.ContainerName, "container", "", "container name of prometheus")
	// cmd.Flags().StringVar(&caPath, "ca-file", "", "path to the CA of TLS enabled cluster")
	// cmd.Flags().StringVar(&certPath, "cert-file", "", "path to the client certification of TLS enabled cluster")
	// cmd.Flags().StringVar(&keyPath, "key-file", "", "path to the private key of client certification of TLS enabled cluster")
	cmd.Flags().StringVarP(&opt.ScrapeBegin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&opt.ScrapeEnd, "to", "t", time.Now().Format(time.RFC3339), "stop timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&cOpt.Dir, "output", "o", "", "output directory of collected data")

	cobra.MarkFlagRequired(cmd.Flags(), "kubeconfig")
	cobra.MarkFlagRequired(cmd.Flags(), "name")
	// cobra.MarkFlagRequired(cmd.Flags(), "cluster-id")
	cobra.MarkFlagRequired(cmd.Flags(), "namespace")
	// cobra.MarkFlagRequired(cmd.Flags(), "pod")
	// cobra.MarkFlagRequired(cmd.Flags(), "container")

	return cmd
}

// buildTopoForK8sCluster creates an abstract topo from tiup-cluster metadata
func bbuildTopoForK8sCluster(
	clusterName string,
	namespace string,
	cfg *rest.Config,
) (clusterID,
	podName string,
	err error) {
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to get kubernetes Clientset: %v", err)
	}
	dynCli, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to get kubernetes dynamic client interface: %v", err)
	}

	gvrMonitor := schema.GroupVersionResource{
		Group:    "pingcap.com",
		Version:  "v1alpha1",
		Resource: "tidbmonitors",
	}

	monList, err := dynCli.Resource(gvrMonitor).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		msg := fmt.Sprintf("failed to list tidbmonitors in namespace %s: %v", namespace, err)
		klog.Errorf(msg)
		return
	}
	monData, err := monList.MarshalJSON()
	if err != nil {
		msg := fmt.Sprintf("failed to marshal tidbmonitors to json: %v", err)
		klog.Errorf(msg)
		return
	}
	var mon pingcapv1alpha1.TidbMonitorList
	if err := json.Unmarshal(monData, &mon); err != nil {
		return "", "", err
	}
	if len(mon.Items) == 0 {
		return "", "", fmt.Errorf("no tidbmonitors found in namespace '%s'", namespace)
	}

	// find monitor pod
	var matchedMon pingcapv1alpha1.TidbMonitor
	monMatched := false
	for _, m := range mon.Items {
		for _, clsRef := range m.Spec.Clusters {
			if clsRef.Name == clusterName {
				monMatched = true
				break
			}
		}
		matchedMon = m
		break
	}

	// get monitor pod
	if monMatched {
		labels := &metav1.LabelSelector{
			MatchLabels: map[string]string{
				label.ManagedByLabelKey: "tidb-operator",
				label.NameLabelKey:      "tidb-cluster",
				label.ComponentLabelKey: "monitor",
				label.InstanceLabelKey:  matchedMon.Name,
			},
		}
		selector, err := metav1.LabelSelectorAsSelector(labels)
		if err != nil {
			return "", "", err
		}
		pods, err := kubeCli.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		return pods.Items[0].Labels[label.ClusterIDLabelKey], pods.Items[0].Name, nil
	}

	return "", "", fmt.Errorf("cannot found monitor pod")
}
