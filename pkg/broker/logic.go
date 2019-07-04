package broker

import (
	"encoding/json"
	"fmt"
	"github.com/arugaki/osb-starter-pack/pkg/dao"
	"github.com/arugaki/osb-starter-pack/pkg/kubernetes"
	"github.com/arugaki/osb-starter-pack/pkg/service"
	"github.com/golang/glog"
	"net/http"
	"sync"

	"github.com/pmorie/osb-broker-lib/pkg/broker"

	osb "github.com/pmorie/go-open-service-broker-client/v2"
)

// BusinessLogic provides an implementation of the broker.BusinessLogic
// interface.
type BusinessLogic struct {
	// Indicates if the broker should handle the requests asynchronously.
	async bool
	// Synchronize go routines.
	sync.RWMutex
	// Catalog Infomations
	catalogs []osb.Service
	// Kubectl apply -f template
	serviceTemplates map[string][]byte
	// serviceId - serviceName mapping
	serivceIdName map[string]string
	// serviceId planId mapping
	serviceIdPlan map[string]map[string]osb.Plan
	// mysql db
	db *dao.Dao
	// kubernetes client
	kcl *kubernetes.KubeCli
	// services
	services map[string]service.Service
}

func (b *BusinessLogic) GetCatalog(c *broker.RequestContext) (*broker.CatalogResponse, error) {
	// Your catalog business logic goes here
	response := &broker.CatalogResponse{}
	osbResponse := &osb.CatalogResponse{
		Services: b.catalogs,
	}

	response.CatalogResponse = *osbResponse
	return response, nil
}

func (b *BusinessLogic) Provision(request *osb.ProvisionRequest, c *broker.RequestContext) (*broker.ProvisionResponse, error) {
	instance, err := b.db.SelectInstance(request.InstanceID)
	if err != nil {
		glog.Errorf("select instance by instance id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusServiceUnavailable,
			ResponseError: err,
		}
	}

	if instance.InstanceID == request.InstanceID {
		description := fmt.Sprintf("instance id %s is exist", request.InstanceID)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:  http.StatusConflict,
			Description: &description,
		}
	}

	serviceName, err := b.getServiceName(request.ServiceID)
	if err != nil {
		glog.Errorf("get service name by service id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	plan, err := b.getPlan(request.ServiceID, request.PlanID)
	if err != nil {
		glog.Errorf("get plan by serivce id and plan id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	srcTemplate, err := b.getServiceTemplate(serviceName)
	if err != nil {
		glog.Errorf("get service template by serivce name failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	namespace, err := getNamespace(request.Parameters)
	if err != nil {
		glog.Errorf("get namespace from parameters failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	instanceName, err := getInstanceName(request.Parameters)
	if err != nil {
		glog.Errorf("get instance name from parameters failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	templateAfterInit, err := templateInit(srcTemplate, namespace, instanceName, instance.InstanceID)
	if err != nil {
		glog.Errorf("apply namespace to templates failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	templateAfterParams, err := b.applyParameters(serviceName, templateAfterInit, request.Parameters)
	if err != nil {
		glog.Errorf("apply parameters to templates failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	templateAfterPlan, err := b.applyPlan(serviceName, templateAfterParams, plan)
	if err != nil {
		glog.Errorf("apply plan to templates failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	params, err := json.Marshal(request.Parameters)
	if err != nil {
		glog.Errorf("apply plan to templates failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	kubeServices, err := b.kcl.CreateService(templateAfterPlan)
	if err != nil {
		glog.Errorf("create services in kubernetes failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	templateFinish, err := b.applySpecial(serviceName, templateAfterPlan, kubeServices)
	if err != nil {
		glog.Errorf("apply special variables failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	_, err = b.kcl.CreateInstance(templateFinish)
	if err != nil {
		glog.Errorf("create deployments in kubernetes failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	dashboardURL, err := b.getDashboardURL(serviceName, request.Parameters, kubeServices)
	if err != nil {
		glog.Errorf("get dashboard url failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	instance = &dao.Instance{
		InstanceID:       request.InstanceID,
		ServiceID:        request.ServiceID,
		InstanceName:     instanceName,
		ServiceName:      serviceName,
		SpaceGUID:        request.SpaceGUID,
		OrganizationGUID: request.OrganizationGUID,
		PlanID:           request.PlanID,
		Namespace:        namespace,
		Parameters:       string(params),
		Yaml:             templateFinish,
	}

	_, err = b.db.InsertInstance(instance)
	if err != nil {
		glog.Errorf("insert into instance failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusServiceUnavailable,
			ResponseError: err,
		}
	}

	response := broker.ProvisionResponse{}
	response.Async = true
	response.DashboardURL = &dashboardURL
	response.OperationKey = succeed()
	return &response, nil
}

func (b *BusinessLogic) Deprovision(request *osb.DeprovisionRequest, c *broker.RequestContext) (*broker.DeprovisionResponse, error) {
	instance, err := b.db.SelectInstance(request.InstanceID)
	if err != nil {
		glog.Errorf("select instance by instance id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusServiceUnavailable,
			ResponseError: err,
		}
	}

	if instance.InstanceID == "" {
		description := fmt.Sprintf("instance id %s is gone", request.InstanceID)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:  http.StatusGone,
			Description: &description,
		}
	}

	err = b.beforeKubeDelete(instance)
	if err != nil {
		glog.Errorf("delete before kubernetes resources failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	err = b.kcl.DeleteInstance(instance.Yaml)
	if err != nil {
		glog.Errorf("delete kubernetes resources failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	err = b.afterKubeDelete(instance)
	if err != nil {
		glog.Errorf("delete before kubernetes resources failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	_, err = b.db.DeleteInstance(request.InstanceID)
	if err != nil {
		glog.Errorf("delete instance by instance id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusServiceUnavailable,
			ResponseError: err,
		}
	}

	response := broker.DeprovisionResponse{}
	response.Async = false
	response.OperationKey = succeed()
	return &response, nil
}

func (b *BusinessLogic) LastOperation(request *osb.LastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	instance, err := b.db.SelectInstance(request.InstanceID)
	if err != nil {
		glog.Errorf("select instance by instance id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusServiceUnavailable,
			ResponseError: err,
		}
	}

	if instance.InstanceID == "" {
		description := fmt.Sprintf("instance id %s is gone", request.InstanceID)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:  http.StatusGone,
			Description: &description,
		}
	}

	response := &broker.LastOperationResponse{}

	podCreating, _, podFailed, err := b.kcl.CheckInstance(instance.InstanceID, instance.Namespace)
	if err != nil {
		glog.Errorf("get deployment status from kubernetes failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	if podFailed {
		response.State = LastStateFailed
		return response, nil
	}

	if podCreating {
		response.State = LastStateProcessing
		return response, nil
	}

	processing, _, failed, err := b.lastStateCheck(instance)
	if err != nil {
		glog.Errorf("last state check failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	if processing {
		response.State = LastStateProcessing
		return response, nil
	}

	if failed {
		response.State = LastStateFailed
		return response, nil
	}

	response.State = LastStateSuccess
	return response, nil
}

func (b *BusinessLogic) Update(request *osb.UpdateInstanceRequest, c *broker.RequestContext) (*broker.UpdateInstanceResponse, error) {
	instance, err := b.db.SelectInstance(request.InstanceID)
	if err != nil {
		glog.Errorf("select instance by instance id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusServiceUnavailable,
			ResponseError: err,
		}
	}

	if instance.InstanceID == request.InstanceID {
		description := fmt.Sprintf("instance id %s is exist", request.InstanceID)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:  http.StatusConflict,
			Description: &description,
		}
	}

	serviceName, err := b.getServiceName(request.ServiceID)
	if err != nil {
		glog.Errorf("get service name by service id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	plan, err := b.getPlan(request.ServiceID, *request.PlanID)
	if err != nil {
		glog.Errorf("get plan by serivce id and plan id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	srcTemplate, err := b.getServiceTemplate(serviceName)
	if err != nil {
		glog.Errorf("get service template by serivce name failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	namespace, err := getNamespace(request.Parameters)
	if err != nil {
		glog.Errorf("get namespace from parameters failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	instanceName, err := getInstanceName(request.Parameters)
	if err != nil {
		glog.Errorf("get instance name from parameters failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	templateAfterInit, err := templateInit(srcTemplate, namespace, instanceName, instance.InstanceID)
	if err != nil {
		glog.Errorf("apply namespace to templates failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	templateAfterParams, err := b.applyParameters(serviceName, templateAfterInit, request.Parameters)
	if err != nil {
		glog.Errorf("apply parameters to templates failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	templateAfterPlan, err := b.applyPlan(serviceName, templateAfterParams, plan)
	if err != nil {
		glog.Errorf("apply plan to templates failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	params, err := json.Marshal(request.Parameters)
	if err != nil {
		glog.Errorf("apply plan to templates failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	kubeServices, err := b.kcl.UpdateService(instance.InstanceID, templateAfterPlan)
	if err != nil {
		glog.Errorf("update services in kubernetes failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	templateFinish, err := b.applySpecial(serviceName, templateAfterPlan, kubeServices)
	if err != nil {
		glog.Errorf("apply special variables failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	_, err = b.kcl.UpdateInstance(instance.InstanceID, templateFinish)
	if err != nil {
		glog.Errorf("create deployments in kubernetes failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	instance = &dao.Instance{
		InstanceID:  request.InstanceID,
		ServiceID:   request.ServiceID,
		ServiceName: serviceName,
		PlanID:      *request.PlanID,
		Namespace:   namespace,
		Parameters:  string(params),
		Yaml:        templateFinish,
	}

	_, err = b.db.UpdateInstance(instance)
	if err != nil {
		glog.Errorf("update instance by instance id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusServiceUnavailable,
			ResponseError: err,
		}
	}

	response := broker.UpdateInstanceResponse{}
	response.Async = true
	response.OperationKey = succeed()
	return &response, nil
}

func (b *BusinessLogic) Bind(request *osb.BindRequest, c *broker.RequestContext) (*broker.BindResponse, error) {
	serviceName, err := b.getServiceName(request.ServiceID)
	if err != nil {
		glog.Errorf("get service name by service id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	cred, err := b.bindInstance(serviceName, request)
	if err != nil {
		glog.Errorf("bind instance failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	response := &broker.BindResponse{}
	response.Credentials = cred
	response.OperationKey = succeed()
	response.Async = false
	return response, nil
}

func (b *BusinessLogic) Unbind(request *osb.UnbindRequest, c *broker.RequestContext) (*broker.UnbindResponse, error) {
	serviceName, err := b.getServiceName(request.ServiceID)
	if err != nil {
		glog.Errorf("get service name by service id failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusBadRequest,
			ResponseError: err,
		}
	}

	err = b.unbindInstance(serviceName, request)
	if err != nil {
		glog.Errorf("unbind instance failed, err is %+v", err)
		return nil, osb.HTTPStatusCodeError{
			StatusCode:    http.StatusInternalServerError,
			ResponseError: err,
		}
	}

	response := &broker.UnbindResponse{}
	response.Async = false
	return response, nil
}

func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
	return nil
}
