package kubetls

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

// Namespace is a newtype of a string
type Namespace string

type clientConfig struct {
	clusterDomain string
	headlessSvc   bool // use headless service to connect, default to use service

	// clientURL is PD/Etcd addr. If it is empty, will generate from target TC
	clientURL string
	// clientKey is client name. If it is empty, will generate from target TC
	clientKey string

	tlsEnable          bool
	tlsSecretNamespace Namespace
	tlsSecretName      string
}

// Option configures the PDClient
type Option func(c *clientConfig)

// ClusterRef sets the cluster domain of TC, it is used when generating the client address from TC.
func ClusterRef(clusterDomain string) Option {
	return func(c *clientConfig) {
		c.clusterDomain = clusterDomain
	}
}

// TLSCertFromTC indicates that the clients use certs from specified TC's secret.
func TLSCertFromTC(ns Namespace, tcName string) Option {
	return func(c *clientConfig) {
		c.tlsSecretNamespace = ns
		c.tlsSecretName = ClusterClientTLSSecretName(tcName)
	}
}

// TLSCertFromTC indicates that clients use certs from specified secret.
func TLSCertFromSecret(ns Namespace, secret string) Option {
	return func(c *clientConfig) {
		c.tlsSecretNamespace = ns
		c.tlsSecretName = secret
	}
}

// getTLSConfig returns *tls.Config for given TiDB cluster.
func getTLSConfig(secretLister corelisterv1.SecretLister, namespace, secretName string) (*tls.Config, error) {
	secret, err := secretLister.Secrets(string(namespace)).Get(secretName)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", namespace, secretName, err)
	}

	return LoadTlsConfigFromSecret(secret)
}

//GetKubeTLSConfig  return *tls.Config for given TiDB cluster on kube
func GetKubeTLSConfig(c kubernetes.Interface, namespace, clusterName string, resync time.Duration) (*tls.Config, error) {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(c, resync)
	secretInformer := kubeInformerFactory.Core().V1().Secrets()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go kubeInformerFactory.Start(ctx.Done())

	return getTLSConfig(secretInformer.Lister(), namespace, ClusterClientTLSSecretName(clusterName))
}
