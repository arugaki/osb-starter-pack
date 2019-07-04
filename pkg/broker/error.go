package broker

import "errors"

var (
	ServiceNotFound         = errors.New("service id is not found")
	ServiceTemplateNotFound = errors.New("service template is not found")
	PlanNotfound            = errors.New("plan id is not found")
	NamespaceNotFound       = errors.New("namespace is not found")
	InstanceNameNotFound    = errors.New("instance name is not found")
)
