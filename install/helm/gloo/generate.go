package main

import (
	"os"

	"github.com/solo-io/gloo/install/helm/gloo/generate"
	"github.com/solo-io/solo-kit/pkg/utils/log"
)

var (
	valuesOutput          = "install/helm/gloo/values.yaml"
	knativeValuesOutput   = "install/helm/gloo/values-knative.yaml"
	chartTemplate         = "install/helm/gloo/Chart-template.yaml"
	chartOutput           = "install/helm/gloo/Chart.yaml"
)

func main() {
	var version string
	if len(os.Args) < 2 {
		panic("Must provide version as argument")
	} else {
		version = os.Args[1]
	}
	log.Printf("Generating helm files.")
	config, err := generateGlooConfigAndWriteYaml(version)
	if err != nil {
		log.Fatalf("generating values.yaml failed!: %v", err)
	}
	if err := generateKnativeValuesYaml(config, version); err != nil {
		log.Fatalf("generating values-knative.yaml failed!: %v", err)
	}
	if err := generateChartYaml(version); err != nil {
		log.Fatalf("generating Chart.yaml failed!: %v", err)
	}
}

func generateGlooConfigAndWriteYaml(version string) (*generate.Config, error) {
	config, err := generate.ReadGlooValuesTemplate()
	if err != nil {
		return nil, err
	}

	config.Gloo.Deployment.Image.Tag = version
	config.Discovery.Deployment.Image.Tag = version
	config.Gateway.Deployment.Image.Tag = version
	config.GatewayProxy.Deployment.Image.Tag = version
	config.Ingress.Deployment.Image.Tag = version
	config.IngressProxy.Deployment.Image.Tag = version

	err = generate.WriteYaml(config, valuesOutput)
	return config, err
}

func generateKnativeValuesYaml(glooValuesTemplate *generate.Config, version string) error {
	if err := generate.UpdateGlooTemplateWithKnativeTemplate(glooValuesTemplate); err != nil {
		return err
	}

	glooValuesTemplate.Settings.Integrations.Knative.Proxy.Image.Tag = version

	return generate.WriteYaml(&glooValuesTemplate, knativeValuesOutput)
}

func generateChartYaml(version string) error {
	var chart generate.Chart
	if err := generate.ReadYaml(chartTemplate, &chart); err != nil {
		return err
	}

	chart.Version = version

	return generate.WriteYaml(&chart, chartOutput)
}
