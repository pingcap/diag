package utils

import (
	"fmt"
	"net/http"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func RunPortForward(kubeconfig, namespace, podName string, port int) (stopChannel chan struct{}, localPort int, err error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, 0, err
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get kubernetes Clientset: %v", err)
	}

	req := kubeCli.CoreV1().RESTClient().Post().Namespace(namespace).
		Resource("pods").Name(podName).SubResource("portforward")

	readyChannel := make(chan struct{})
	stopChannel = make(chan struct{}, 1)
	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, 0, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	fw, err := portforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{fmt.Sprintf(":%d", port)}, stopChannel, readyChannel, nil, nil)
	if err != nil {
		return nil, 0, err
	}
	go fw.ForwardPorts()
	<-readyChannel

	ports, err := fw.GetPorts()
	if err != nil {
		return nil, 0, err
	}

	return stopChannel, int(ports[0].Local), nil
}
