package service

import (
	"github.com/arugaki/osb-starter-pack/pkg/dao"
	"github.com/pmorie/go-open-service-broker-client/v2"
)

type ZookeeperService struct {}


func (z *ZookeeperService) Name() string {
	return "zookeeper"
}

func (z *ZookeeperService) ApplyParameters(template string, params map[string]interface{}) (string, error) {
	return "", nil
}

func (z *ZookeeperService) ApplyPlan(template string, plan *v2.Plan) (string, error) {
	return "", nil
}

func (z *ZookeeperService) ApplySpecial(template string, kubeServices map[string]string) (string, error) {
	return "", nil
}

func (z *ZookeeperService) GetDashboardURL(params map[string]interface{}, kubeServices map[string]string) (string, error) {
	return "", nil
}

func (z *ZookeeperService) BeforeKubeDelete(instance *dao.Instance) error {
	return nil
}

func (z *ZookeeperService) AfterKubeDelete(instance *dao.Instance) error {
	return nil
}

func (z *ZookeeperService) LastStateCheck(instance *dao.Instance) (bool, bool, bool, error) {
	return false, false, false, nil
}

func (z *ZookeeperService) BindInstance(request *v2.BindRequest) (map[string]interface{}, error) {
	return nil, nil
}

func (z *ZookeeperService) UnbindInstance(request *v2.UnbindRequest) error {
	return nil
}