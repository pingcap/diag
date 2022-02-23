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

package tls

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// GetDmClientTLSConfig return *tls.Config for given DM clinet
func GetDmClientTLSConfig(c kubernetes.Interface, namespace, dcName string, resync time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, dmClientTLSSecretName(dcName), resync)
}

// GetClusterClientClientTLSConfig  GetDmClientTLSConfig return *tls.Config for given tidb cluster clinet
func GetClusterClientTLSConfig(c kubernetes.Interface, namespace, tcName string, resync time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, clusterClientTLSSecretName(tcName), resync)
}

// GetClusterTLSConfig  GetDmClientTLSConfig return *tls.Config for given tidb cluster
func GetClusterTLSConfig(c kubernetes.Interface, namespace, tcName, component string, resync time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, clusterTLSSecretName(tcName, component), resync)
}

// GetTiDBClientTLSConfig  GetDmClientTLSConfig return *tls.Config for given tidb client
func GetTiDBClientTLSConfig(c kubernetes.Interface, namespace, tcName string, resync time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, tiDBClientTLSSecretName(tcName), resync)
}

// GetTiDBServerTLSConfig  GetDmClientTLSConfig return *tls.Config for given tidb client
func GetTiDBServerTLSConfig(c kubernetes.Interface, namespace, tcName string, resync time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, tiDBServerTLSSecretName(tcName), resync)
}

// getKubeTLSConfig  return *tls.Config for given TiDB cluster on kube
func getTLSConfig(c kubernetes.Interface, namespace, secretName string, resync time.Duration) (*tls.Config, error) {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(c, resync)
	secretInformer := kubeInformerFactory.Core().V1().Secrets()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go kubeInformerFactory.Start(ctx.Done())

	// waiting for the shared informer's store has synced.
	cache.WaitForCacheSync(ctx.Done(), secretInformer.Informer().HasSynced)

	secret, err := secretInformer.Lister().Secrets(namespace).Get(secretName)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", namespace, secretName, err)
	}

	return LoadTlsConfigFromSecret(secret)
}
