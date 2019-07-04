package kubernetes

import (
	"fmt"
	"github.com/golang/glog"
	"io"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	discoveryutil "kmodules.xyz/client-go/discovery"
)

func GetKubernetesClient(kubeConfigPath string) (Interface, error) {
	var clientConfig *clientrest.Config
	var err error
	if kubeConfigPath == "" {
		clientConfig, err = clientrest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err := clientcmd.LoadFromFile(kubeConfigPath)
		if err != nil {
			return nil, err
		}

		clientConfig, err = clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return nil, err
		}
	}
	return NewForConfig(clientConfig)
}

type KubeCli struct {
	Client Interface
}

func New(kubeConfig string) (*KubeCli, error) {
	client, err := GetKubernetesClient(kubeConfig)
	if err != nil {
		return nil, err
	}

	return &KubeCli{
		Client: client,
	}, nil
}

func (k *KubeCli) createFromReader(filter func(obj *unstructured.Unstructured) bool, reader io.Reader)(map[string]string, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	needCreateResources := make(map[string]string)
	for {
		// unmarshals the next object from the underlying stream into the provide object
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			glog.Errorf("failed to decode the next object from the underlying stream into an unstructured object: %v", err)
			return nil, err
		}

		// find the object's resource interface
		gvk := obj.GroupVersionKind()
		gvr, err := discoveryutil.ResourceForGVK(k.Client.Discovery(), gvk)
		if err != nil {
			glog.Errorf("failed to discovery GVR for the resource %v: %v", gvk, err)
			return nil, err
		}
		namespace := obj.GetNamespace()
		name := obj.GetName()
		ri := k.Client.Resource(gvr).Namespace(namespace)

		if needCreate := filter(obj.DeepCopy()); !needCreate {
			continue
		}

		// create the object using its resource interface
		obj.SetResourceVersion("")
		_, err = ri.Create(obj)
		if err != nil {
			glog.Errorf("failed to create the resource %s/%s: %v", namespace, name, err)
			return nil, err
		}

		needCreateResources[obj.GetNamespace()] = obj.GetName()
	}
	return needCreateResources, nil
}

func (k *KubeCli) createOrUpdateFromReader(filter func(obj *unstructured.Unstructured) bool, reader io.Reader) (map[string]string, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	needCreateResources := make(map[string]string)
	for {
		// unmarshals the next object from the underlying stream into the provide object
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			glog.Errorf("failed to decode the next object from the underlying stream into an unstructured object: %v", err)
			return nil, err
		}

		// find the object's resource interface
		gvk := obj.GroupVersionKind()
		gvr, err := discoveryutil.ResourceForGVK(k.Client.Discovery(), gvk)
		if err != nil {
			glog.Errorf("failed to discovery GVR for the resource %v: %v", gvk, err)
			return nil, err
		}
		namespace := obj.GetNamespace()
		ri := k.Client.Resource(gvr).Namespace(namespace)

		// handle the object using its resource interface
		name := obj.GetName()
		kind := obj.GetKind()

		if needCreate := filter(obj.DeepCopy()); !needCreate {
			continue
		}

		oldObj, err := ri.Get(name, metav1.GetOptions{})
		if err != nil {
			if !kapierrors.IsNotFound(err) {
				msg := fmt.Sprintf("failed to retrieve current configuration of the %s %s/%s: %v", kind, namespace, name, err)
				glog.Errorln(msg)
				return nil, fmt.Errorf(msg)
			}

			// create it because the resource is not existed
			obj.SetResourceVersion("")
			_, err := ri.Create(obj)
			if err != nil {
				glog.Infof("failed to create the %s resource %s/%s: %v", kind, namespace, name, err)
				return nil, err
			}
			continue
		}
		// found the old resource, so we update it
		obj.SetResourceVersion(oldObj.GetResourceVersion())
		_, err = ri.Update(obj)
		if err != nil {
			glog.Errorf("failed to update the existed %s resource %s/%s, %v", kind, namespace, name, err)
			return nil, err
		}

		needCreateResources[obj.GetNamespace()] = obj.GetName()
	}
	return needCreateResources, nil
}

func  (k *KubeCli) deleteFromReader(reader io.Reader) error {
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			glog.Errorf("failed to decode the next object from the underlying stream into an unstructured object: %v", err)
			return err
		}

		// find the object's resource interface
		gvk := obj.GroupVersionKind()
		gvr, err := discoveryutil.ResourceForGVK(k.Client.Discovery(), gvk)
		if err != nil {
			glog.Errorf("failed to discovery GVR for the resource %v: %v", gvk, err)
			return err
		}

		kind := obj.GetKind()
		namespace := obj.GetNamespace()
		name := obj.GetName()

		ri := k.Client.Resource(gvr).Namespace(namespace)
		err = ri.Delete(name, &metav1.DeleteOptions{})
		if err != nil && !kapierrors.IsNotFound(err) {
			glog.Infof("failed to delete the %s resource %s/%s: %v", kind, namespace, name, err)
			return err
		}
	}
	return nil
}