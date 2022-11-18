package kubernetes

import (
	"fmt"

	"github.com/vshn/provider-cloudscale/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClient creates a k8s client from the server url and token url
func NewClient(kubeconfig, url, token string) (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("core scheme: %w", err)
	}
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("cannot add k8s exoscale scheme: %w", err)
	}

	config := &rest.Config{Host: url, BearerToken: token}

	// kubeconfig takes precedence if set.
	if kubeconfig != "" {
		// use the current context in kubeconfig
		c, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("unable to load from flags: %w", err)
		}
		config = c
	}

	c, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot initialize k8s client: %w", err)
	}
	return c, nil
}
