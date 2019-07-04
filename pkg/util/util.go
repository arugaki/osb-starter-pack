package util

import (
	"bytes"
	"fmt"
	"github.com/pmorie/go-open-service-broker-client/v2"
	"text/template"
)

func ExecuteTemplate(t string, data interface{}) (string, error) {
	tmpl, err := template.New("tmpl").Parse(t)
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

func GetStringParam(params map[string]interface{}, name string) (string, error) {
	if ns, ok := params[name]; ok {
		return fmt.Sprintf("%v", ns), nil
	} else {
		return "", fmt.Errorf("the param %s not found", name)
	}
}

func GetQuotaFromPlan(plan *v2.Plan) (string, string, string, error) {
	if bullets, ok := plan.Metadata["bullets"]; ok {
		quota, ok := bullets.([]string)
		if !ok {
			return "", "", "", fmt.Errorf("unexpects bullets in plan")
		}
		if len(quota) != 3 {
			return "", "", "", fmt.Errorf("unexpects quota in bullets")
		}
		return quota[0], quota[1], quota[2], nil
	}

	return "", "", "", fmt.Errorf("no bullets in plan")
}