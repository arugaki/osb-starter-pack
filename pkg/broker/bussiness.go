package broker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/arugaki/osb-starter-pack/pkg/asset"
	"github.com/arugaki/osb-starter-pack/pkg/dao"
	"github.com/arugaki/osb-starter-pack/pkg/kubernetes"
	"github.com/arugaki/osb-starter-pack/pkg/service"
	"github.com/golang/glog"
	"github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/pmorie/osb-broker-lib/pkg/broker"
	"strings"
	"text/template"
)

const (
	LastStateSuccess    = "succeed"
	LastStateFailed     = "failed"
	LastStateProcessing = "processing"
)

func succeed() *v2.OperationKey {
	var o v2.OperationKey
	o = "succeed"
	return &o
}

var _ broker.Interface = &BusinessLogic{}

// NewBusinessLogic is a hook that is called with the Options the program is run
// with. NewBusinessLogic is the place where you will initialize your
// BusinessLogic the parameters passed in.
func NewBusinessLogic(o Options) (*BusinessLogic, error) {
	b := &BusinessLogic{
		async:            o.Async,
		catalogs:         make([]v2.Service, 0, 10),
		serviceTemplates: make(map[string][]byte),
		serivceIdName:    make(map[string]string),
		serviceIdPlan:    make(map[string]map[string]v2.Plan),
		services:         make(map[string]service.Service),
	}

	b.InitServices()

	err := b.InitServiceCatalog()
	if err != nil {
		glog.Errorf("init service failed, err is %+v", err)
		return nil, err
	}

	c := &dao.Config{
		Addr:     o.MysqlAddress,
		Port:     o.MysqlPort,
		UserName: o.MysqlUserName,
		Password: o.MysqlPassword,
		DB:       o.MysqlDB,
		Active:   o.MysqlActive,
		Idle:     o.MysqlIdle,
	}
	b.db, err = dao.New(c)
	if err != nil {
		glog.Errorf("init dao failed, err is %+v", err)
		return nil, err
	}

	b.kcl, err = kubernetes.New(o.KubeConfig)
	if err != nil {
		glog.Errorf("init kubernetes failed, err is %+v", err)
		return nil, err
	}

	return b, nil
}

func (b *BusinessLogic) InitServices() {
	services := make(map[string]service.Service)

	zookeeperService := &service.ZookeeperService{}

	services[zookeeperService.Name()] = zookeeperService
}

func (b *BusinessLogic) InitServiceCatalog() error {
	catalogs, serviceTemplates, serivceIdName, serviceIdPlan, err := InitServiceTemplate()
	if err != nil {
		return err
	}

	b.catalogs = catalogs
	b.serviceTemplates = serviceTemplates
	b.serivceIdName = serivceIdName
	b.serviceIdPlan = serviceIdPlan
	return nil
}

func InitServiceTemplate() ([]v2.Service, map[string][]byte, map[string]string, map[string]map[string]v2.Plan, error) {

	var catalogs []v2.Service
	serviceTemplates := make(map[string][]byte)
	serivceIdName := make(map[string]string)
	serviceIdPlan := make(map[string]map[string]v2.Plan)

	for _, name := range asset.AssetNames() {
		data, err := asset.Asset(name)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		if strings.Contains(name, "_generated.json") {
			var catalog v2.Service
			err = json.Unmarshal(data, &catalog)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			catalogs = append(catalogs, catalog)
			serivceIdName[catalog.ID] = catalog.Name

			plans := make(map[string]v2.Plan)
			for _, plan := range catalog.Plans {
				plans[plan.ID] = plan
			}
			serviceIdPlan[catalog.ID] = plans

		} else {
			serviceName := strings.Split(name, ".")[0]
			serviceTemplates[serviceName] = data
		}
	}

	return catalogs, serviceTemplates, serivceIdName, serviceIdPlan, nil
}

func (b *BusinessLogic) getServiceName(serviceId string) (string, error) {
	if name, ok := b.serivceIdName[serviceId]; ok {
		return name, nil
	} else {
		return "", ServiceNotFound
	}
}

func (b *BusinessLogic) getPlan(serviceId, planId string) (*v2.Plan, error) {
	if plans, ok := b.serviceIdPlan[serviceId]; ok {
		if plan, ok := plans[planId]; ok {
			return &plan, nil
		} else {
			return nil, PlanNotfound
		}
	} else {
		return nil, ServiceNotFound
	}
}

func (b *BusinessLogic) getServiceTemplate(serviceName string) (string, error) {
	if t, ok := b.serviceTemplates[serviceName]; ok {
		return string(t), nil
	} else {
		return "", ServiceTemplateNotFound
	}
}

func getNamespace(params map[string]interface{}) (string, error) {
	if ns, ok := params["NAMESPACE"]; ok {
		return fmt.Sprintf("%v", ns), nil
	} else {
		return "", NamespaceNotFound
	}
}

func getInstanceName(params map[string]interface{}) (string, error) {
	if ns, ok := params["INSTANCE_NAME"]; ok {
		return fmt.Sprintf("%v", ns), nil
	} else {
		return "", InstanceNameNotFound
	}
}

// replace template instanceid and namespace
func templateInit(template, namespace, instanceName, id string) (string, error) {
	type Params struct {
		Id           string
		Namespace    string
		InstanceName string
	}

	newTemplate, err := executeTemplate(template, Params{
		Id:           id,
		Namespace:    namespace,
		InstanceName: instanceName,
	})
	if err != nil {
		return "", err
	}

	return newTemplate, nil
}

func executeTemplate(t string, data interface{}) (string, error) {
	tmpl, err := template.New("service").Parse(t)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), err
}

func (b *BusinessLogic) applyParameters(serviceName, template string, params map[string]interface{}) (string, error) {
	if s, ok := b.services[serviceName]; ok {
		t, err := s.ApplyParameters(template, params)
		if err != nil {
			return "", err
		}
		return t, nil
	}

	return "", ServiceNotFound
}

func (b *BusinessLogic) applyPlan(serviceName, template string, plan *v2.Plan) (string, error) {
	if s, ok := b.services[serviceName]; ok {
		t, err := s.ApplyPlan(template, plan)
		if err != nil {
			return "", err
		}
		return t, nil
	}

	return "", ServiceNotFound
}

func (b *BusinessLogic) applySpecial(serviceName, template string, kubeServices map[string]string) (string, error) {
	if s, ok := b.services[serviceName]; ok {
		t, err := s.ApplySpecial(template, kubeServices)
		if err != nil {
			return "", err
		}
		return t, nil
	}

	return "", ServiceNotFound
}

func (b *BusinessLogic) getDashboardURL(serviceName string, params map[string]interface{}, kubeServices map[string]string) (string, error) {
	if s, ok := b.services[serviceName]; ok {
		url, err := s.GetDashboardURL(params, kubeServices)
		if err != nil {
			return "", err
		}
		return url, nil
	}

	return "", ServiceNotFound
}

func (b *BusinessLogic) beforeKubeDelete(instance *dao.Instance) error {
	if s, ok := b.services[instance.ServiceName]; ok {
		err := s.BeforeKubeDelete(instance)
		if err != nil {
			return err
		}
		return nil
	}

	return ServiceNotFound
}

func (b *BusinessLogic) afterKubeDelete(instance *dao.Instance) error {
	if s, ok := b.services[instance.ServiceName]; ok {
		err := s.AfterKubeDelete(instance)
		if err != nil {
			return err
		}
		return nil
	}

	return ServiceNotFound
}

func (b *BusinessLogic) lastStateCheck(instance *dao.Instance) (bool, bool, bool, error) {
	if s, ok := b.services[instance.ServiceName]; ok {
		processing, success, failed, err := s.LastStateCheck(instance)
		if err != nil {
			return false, false, false, err
		}
		return processing, success, failed, nil
	}

	return false, false, false, ServiceNotFound
}

func (b *BusinessLogic) bindInstance(serviceName string, request *v2.BindRequest) (map[string]interface{}, error) {
	if s, ok := b.services[serviceName]; ok {
		creds, err := s.BindInstance(request)
		if err != nil {
			return nil, err
		}
		return creds, nil
	}

	return nil, ServiceNotFound
}

func (b *BusinessLogic) unbindInstance(serviceName string, request *v2.UnbindRequest) error {
	if s, ok := b.services[serviceName]; ok {
		err := s.UnbindInstance(request)
		if err != nil {
			return err
		}
		return nil
	}

	return ServiceNotFound
}
