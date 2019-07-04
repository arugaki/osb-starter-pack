package service

import (
	"github.com/arugaki/osb-starter-pack/pkg/dao"
	"github.com/arugaki/osb-starter-pack/pkg/kubernetes"
	"github.com/pmorie/go-open-service-broker-client/v2"
)

type Service interface {
	// 返回该服务的名字, 与模版中的一致
	Name() string
	// 根据参数替换go template 中的变量
	// 替换 plan.parametes
	ApplyParameters(template string, params map[string]interface{}) (string, error)
	// 替换 plan.bulletes.quota
	ApplyPlan(template string, plan *v2.Plan) (string, error)
	// 在部署了 service 之后执行, kubeService 为已部署的 service namespace-serviceName 映射
	ApplySpecial(template string, kubeServices map[string]string, Client kubernetes.Interface) (string, error)
	// 得到服务 web console 的 url
	GetDashboardURL(params map[string]interface{}, kubeServices map[string]string, kcl kubernetes.Interface) (string, error)
	// 自定义在删除kubernetes的资源前的操作
	BeforeKubeDelete(instance *dao.Instance) error
	// 自定义在删除kubernetes的资源后的操作
	AfterKubeDelete(instance *dao.Instance) error
	// 自定义服务创建成功的检查
	LastStateCheck(instance *dao.Instance) (bool, bool, bool, error)

	BindInstance(request *v2.BindRequest) (map[string]interface{}, error)
	UnbindInstance(request *v2.UnbindRequest) error
}