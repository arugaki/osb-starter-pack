package service

import (
	"github.com/arugaki/osb-starter-pack/pkg/dao"
	"github.com/pmorie/go-open-service-broker-client/v2"
)

type Service interface {
	Name() string

	ApplyParameters(template string, params map[string]interface{}) (string, error)
	ApplyPlan(template string, plan *v2.Plan) (string, error)
	ApplySpecial(template string, kubeServices map[string]string) (string, error)
	GetDashboardURL(params map[string]interface{}, kubeServices map[string]string) (string, error)

	BeforeKubeDelete(instance *dao.Instance) error
	AfterKubeDelete(instance *dao.Instance) error

	LastStateCheck(instance *dao.Instance) (bool, bool, bool, error)

	BindInstance(request *v2.BindRequest) (map[string]interface{}, error)

	UnbindInstance(request *v2.UnbindRequest) error
}