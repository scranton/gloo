package generate

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

var (
	knativeValuesTemplate = "install/helm/gloo/values-knative-template.yaml"
	valuesTemplate        = "install/helm/gloo/values-template.yaml"
)

func ReadYaml(path string, obj interface{}) error {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "failed reading server config file: %s", path)
	}

	if err := yaml.Unmarshal(bytes, obj); err != nil {
		return errors.Wrap(err, "failed parsing configuration file")
	}

	return nil
}

func ReadGlooValuesTemplate() (*Config, error) {
	var glooValuesTemplate Config
	if err := ReadYaml(valuesTemplate, &glooValuesTemplate); err != nil {
		return nil, err
	}
	return &glooValuesTemplate, nil
}

func UpdateGlooTemplateWithKnativeTemplate(glooValuesTemplate *Config) error {
	var glooKnativeValuesTemplate Config
	if err := ReadYaml(knativeValuesTemplate, &glooKnativeValuesTemplate); err != nil {
		return err
	}

	glooValuesTemplate.Settings.Integrations.Knative = glooKnativeValuesTemplate.Settings.Integrations.Knative
	return nil
}

