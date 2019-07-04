package kubernetes

import (
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubeCli) CreateService(yaml string) (map[string]string, error) {
	filter := func(obj *unstructured.Unstructured) bool {
		kind := obj.GetKind()
		if kind == "Service" {
			return true
		}
		return false
	}

	kubeServices, err := k.createFromReader(filter, strings.NewReader(yaml))
	if err != nil {
		return nil, err
	}
	return kubeServices, nil
}

func (k *KubeCli) CreateInstance(yaml string) (map[string]string, error) {
	filter := func(obj *unstructured.Unstructured) bool {
		kind := obj.GetKind()
		if kind != "Service" && kind != "Ingress" {
			return true
		}
		return false
	}

	kubeDeployments, err := k.createFromReader(filter, strings.NewReader(yaml))
	if err != nil {
		return nil, err
	}
	return kubeDeployments, nil
}

func (k *KubeCli) DeleteInstance(yaml string) error {
	err := k.deleteFromReader(strings.NewReader(yaml))
	if err != nil {
		return err
	}
	return nil
}

func (k *KubeCli) CheckInstance(id, namespace string) (bool, bool, bool, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("ruyiyun.servicebroker/instance=%s", id),
	}

	pods, err := k.Client.CoreV1().Pods(namespace).List(listOptions)
	if err != nil {
		glog.Errorf("failed to retrieve related Deployment with instance id %q: %v", id, err)
		return false, false, false, err
	}

	ready, failed, sum := 0, 0, 0
	for _, pod := range pods.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			sum++
			if containerStatus.Ready {
				ready++
				continue
			}
			if containerStatus.State.Waiting.Reason != "ContainerCreating" {
				failed++
			}
		}
	}

	if failed != 0 {
		return false, false, true, nil
	}
	if ready != sum {
		return false, true, false, nil
	}
	return true, false, false, nil
}

func (k *KubeCli) UpdateService(instanceId, yaml string) (map[string]string, error) {
	filter := func(obj *unstructured.Unstructured) bool {
		kind := obj.GetKind()
		if kind == "Service" {
			return true
		}
		return false
	}

	kubeServices, err := k.createOrUpdateFromReader(filter, strings.NewReader(yaml))
	if err != nil {
		return nil, err
	}
	return kubeServices, nil
}

func (k *KubeCli) UpdateInstance(instanceId, yaml string) (map[string]string, error) {
	filter := func(obj *unstructured.Unstructured) bool {
		kind := obj.GetKind()
		if kind != "Service" && kind != "Ingress" && kind != "Router"{
			return true
		}
		return false
	}

	kubeDeployments, err := k.createOrUpdateFromReader(filter, strings.NewReader(yaml))
	if err != nil {
		return nil, err
	}
	return kubeDeployments, nil
}