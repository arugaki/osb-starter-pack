package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Interface defines the external client interface for kubernetes and openshift cluster.
type Interface interface {
	kubernetes.Interface
	dynamic.Interface
	ClientConfig() *rest.Config
}

// make sure that a Clientset instance implement the interface.
var _ = Interface(&Clientset{})

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	config *rest.Config
	*kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// Resource implement kuberentes dynamic interface
func (c *Clientset) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return c.dynamicClient.Resource(resource)
}

// ClientConfig returns a complete client config
func (c *Clientset) ClientConfig() *rest.Config {
	if c == nil {
		return nil
	}
	return c.config
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	var sc Clientset
	var err error

	sc.config = c

	sc.Clientset, err = kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	sc.dynamicClient, err = dynamic.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return &sc, nil
}