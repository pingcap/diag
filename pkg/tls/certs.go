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
	"crypto/tls"
	"crypto/x509"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func dmClientTLSSecretName(dcName string) string {
	return fmt.Sprintf("%s-dm-client-secret", dcName)
}

func clusterClientTLSSecretName(tcName string) string {
	return fmt.Sprintf("%s-cluster-client-secret", tcName)
}

func clusterTLSSecretName(tcName, component string) string {
	return fmt.Sprintf("%s-%s-cluster-secret", tcName, component)
}

func tiDBClientTLSSecretName(tcName string) string {
	return fmt.Sprintf("%s-tidb-client-secret", tcName)
}

func tiDBServerTLSSecretName(tcName string) string {
	return fmt.Sprintf("%s-tidb-server-secret", tcName)
}

func LoadTLSConfigFromSecret(secret *corev1.Secret) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()
	var tlsCert tls.Certificate

	if !rootCAs.AppendCertsFromPEM(secret.Data[corev1.ServiceAccountRootCAKey]) {
		return nil, fmt.Errorf("failed to append ca certs")
	}

	clientCert, certExists := secret.Data[corev1.TLSCertKey]
	clientKey, keyExists := secret.Data[corev1.TLSPrivateKeyKey]
	if !certExists || !keyExists {
		return nil, fmt.Errorf("cert or key does not exist in secret %s/%s", secret.Namespace, secret.Name)
	}
	tlsCert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", secret.Namespace, secret.Name, err)
	}

	return &tls.Config{
		RootCAs:      rootCAs,
		ClientCAs:    rootCAs,
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{tlsCert},
	}, nil
}
