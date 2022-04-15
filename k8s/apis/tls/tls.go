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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetDmClientTLSConfig return *tls.Config for given DM clinet
func GetDmClientTLSConfig(c kubernetes.Interface, namespace, dcName string, timemot time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, dmClientTLSSecretName(dcName), timemot)
}

// GetClusterClientClientTLSConfig  GetDmClientTLSConfig return *tls.Config for given tidb cluster clinet
func GetClusterClientTLSConfig(c kubernetes.Interface, namespace, tcName string, timemot time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, clusterClientTLSSecretName(tcName), timemot)
}

// GetClusterTLSConfig  GetDmClientTLSConfig return *tls.Config for given tidb cluster
func GetClusterTLSConfig(c kubernetes.Interface, namespace, tcName, component string, timemot time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, clusterTLSSecretName(tcName, component), timemot)
}

// GetTiDBClientTLSConfig  GetDmClientTLSConfig return *tls.Config for given tidb client
func GetTiDBClientTLSConfig(c kubernetes.Interface, namespace, tcName string, timemot time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, tiDBClientTLSSecretName(tcName), timemot)
}

// GetTiDBServerTLSConfig  GetDmClientTLSConfig return *tls.Config for given tidb client
func GetTiDBServerTLSConfig(c kubernetes.Interface, namespace, tcName string, timemot time.Duration) (*tls.Config, error) {
	return getTLSConfig(c, namespace, tiDBServerTLSSecretName(tcName), timemot)
}

// getTLSConfig  return *tls.Config for given TiDB cluster on kube
func getTLSConfig(c kubernetes.Interface, namespace, secretName string, timemot time.Duration) (*tls.Config, error) {
	// set timemot
	ctx, cancel := context.WithTimeout(context.Background(), timemot)
	defer cancel()

	secret, err := c.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", namespace, secretName, err)
	}
	return LoadTlsConfigFromSecret(secret)
}
