package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pmorie/go-open-service-broker-client/v2"
	"io/ioutil"
	"os"
)

type TemplateConfig struct {
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Tags                []string               `json:"tags"`
	PlanUpdateable      *bool                  `json:"plan_updateable"`
	Bindable            bool                   `json:"bindable"`
	BindingsRetrievable bool                   `json:"bindings_retrievable"`
	Metadata            map[string]interface{} `json:"metadata"`
	CpuQuota            []string               `json:"cpu_quota"`
	MemoryQuota         []string               `json:"memory_quota"`
	DiskQuota           []string               `json:"disk_quota"`
	GpuQuota            []string               `json:"gpu_quota"`
	Properties          map[string]Property    `json:"properties"`
}

type Property struct {
	Description string `json:"description"`
	Default     string `json:"default"`
	Required    bool   `json:"required"`
}

var options struct {
	InputPath  string
	OutputPath string
}

func init() {
	flag.StringVar(&options.InputPath, "in", "", "use '--in' option to specify the path where the service template config in")
	flag.StringVar(&options.OutputPath, "out", "", "use '--out' option to specify the path where the service info out")
	flag.Parse()
}

func truePtr() *bool {
	b := true
	return &b
}

func main() {

	if options.InputPath == "" || options.OutputPath == "" {
		options.InputPath = "/Users/arugaki/hub/osb-starter-pack/src/github.com/pmorie/osb-starter-pack/template"
		options.OutputPath = "/Users/arugaki/hub/osb-starter-pack/src/github.com/pmorie/osb-starter-pack/pkg/asset/template/catalog"
		//flag.Usage()
	}

	templateConfigs, err := loadTemplate(options.InputPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	templates := generateTemplate(templateConfigs)

	err = writeTemplate(options.OutputPath, templates)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadTemplate(path string) (map[string]TemplateConfig, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	templates := make(map[string]TemplateConfig)
	for _, f := range files {
		var tc TemplateConfig
		file, err := ioutil.ReadFile(path + "/" + f.Name())
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(file, &tc)
		if err != nil {
			return nil, err
		}

		templates[tc.Name] = tc
	}

	return templates, nil
}

func generateTemplate(templateConfigs map[string]TemplateConfig) map[string]v2.Service {
	templates := make(map[string]v2.Service)
	for name, templateConfig := range templateConfigs {
		service := v2.Service{}
		service.Name = templateConfig.Name
		service.Description = templateConfig.Description
		service.Tags = templateConfig.Tags
		service.Bindable = templateConfig.Bindable
		service.BindingsRetrievable = templateConfig.BindingsRetrievable
		service.PlanUpdatable = templateConfig.PlanUpdateable
		service.Metadata = templateConfig.Metadata
		service.ID = uuid()

		plans := func(templateConfig TemplateConfig) []v2.Plan {
			plans := make([]v2.Plan, 0, 512)
			for _, cpu := range templateConfig.CpuQuota {
				for _, memory := range templateConfig.MemoryQuota {
					for _, disk := range templateConfig.DiskQuota {
						for _, gpu := range templateConfig.GpuQuota {
							plan := v2.Plan{}
							plan.Description = ""
							plan.Name = fmt.Sprintf("p-%s-%s-%s-%s", cpu, memory, disk, gpu)
							plan.Bindable = truePtr()
							plan.Free = truePtr()
							plan.ID = uuid()
							plan.Metadata = map[string]interface{}{
								"need_quota": true,
								"bullets":    []string{cpu, memory, disk, gpu},
							}
							schemas := generateSchemas(templateConfig.Properties)
							plan.Schemas = schemas

							plans = append(plans, plan)
						}
					}
				}
			}
			return plans
		}(templateConfig)
		service.Plans = plans

		templates[name] = service
	}
	return templates
}

func generateSchemas(properties map[string]Property) *v2.Schemas {
	schemas := v2.Schemas{}
	serviceInstance := v2.ServiceInstanceSchema{}
	serviceBinding := v2.ServiceBindingSchema{}

	params := v2.InputParametersSchema{
		Parameters: properties,
	}

	bindingParams := v2.RequestResponseSchema{}
	bindingParams.InputParametersSchema = params

	serviceInstance.Create = &params
	serviceInstance.Update = &params
	serviceBinding.Create = &bindingParams

	schemas.ServiceInstance = &serviceInstance
	schemas.ServiceBinding = &serviceBinding
	return &schemas
}

func writeTemplate(path string, templates map[string]v2.Service) error {
	for name, template := range templates {
		content, err := json.MarshalIndent(template, "", "  ")
		if err != nil {
			return err
		}
		fileName := fmt.Sprintf("%s/%s_generated.json", path, name)
		err = ioutil.WriteFile(fileName, content, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func uuid() string {
	file, _ := os.Open("/dev/urandom")
	b := make([]byte, 16)
	file.Read(b)
	file.Close()
	uuid := fmt.Sprintf("%x", b)
	return uuid
}