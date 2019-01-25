package helpers

import (
	"fmt"
	"github.com/solo-io/gloo/install/helm/gloo/generate"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/solo-io/solo-kit/pkg/errors"
	"github.com/solo-io/solo-kit/pkg/utils/log"
	"github.com/solo-io/solo-kit/test/setup"
)

func DeployGlooWithHelm(namespace, imageVersion string, enableKnative, verbose bool) error {
	log.Printf("deploying gloo with version %v", imageVersion)
	values, err := ioutil.TempFile("", "gloo-test-")
	if err != nil {
		return err
	}
	defer os.Remove(values.Name())
	if err := WriteGlooHelmValues(values, namespace, imageVersion, enableKnative); err != nil {
		return err
	}
	if err = values.Close(); err != nil {
		return err
	}

	// make the manifest
	manifestContents, err := RunCommandOutput(verbose,
		"helm", "template", GlooHelmChartDir(),
		"--namespace", namespace,
		"-f", values.Name(),
	)
	if err != nil {
		return err
	}

	if err := RunCommandInput(manifestContents, verbose, "kubectl", "apply", "-f", "-"); err != nil {
		return err
	}

	return nil
}

func WriteGlooHelmValues(file *os.File, namespace, version string, enableKnative bool) error {
	glooValuesTemplate, err := generate.ReadGlooValuesTemplate()
	if err != nil {
		return err
	}

	glooValuesTemplate.Discovery.Deployment.Image.Tag = version
	glooValuesTemplate.Gateway.Deployment.Image.Tag = version
	glooValuesTemplate.GatewayProxy.Deployment.Image.Tag = version
	glooValuesTemplate.Gloo.Deployment.Image.Tag = version
	glooValuesTemplate.Ingress.Deployment.Image.Tag = version
	glooValuesTemplate.IngressProxy.Deployment.Image.Tag = version
	glooValuesTemplate.Settings.WriteNamespace = namespace

	if enableKnative {
		generate.UpdateGlooTemplateWithKnativeTemplate(glooValuesTemplate)
		glooValuesTemplate.Settings.Integrations.Knative.Proxy.Image.Tag = version
	}

	generate.WriteYaml(glooValuesTemplate, file.Name())

	return nil
}

var glooPodLabels = []string{
	"gloo=gloo",
	"gloo=discovery",
	"gloo=gateway",
	"gloo=ingress",
}

func WaitGlooPods(timeout, interval time.Duration) error {
	if err := WaitPodsRunning(timeout, interval, glooPodLabels...); err != nil {
		return err
	}
	return nil
}

func WaitPodsRunning(timeout, interval time.Duration, labels ...string) error {
	finished := func(output string) bool {
		return strings.Contains(output, "Running") || strings.Contains(output, "ContainerCreating")
	}
	for _, label := range labels {
		if err := WaitPodStatus(timeout, interval, label, "Running", finished); err != nil {
			return err
		}
	}
	finished = func(output string) bool {
		return strings.Contains(output, "Running")
	}
	for _, label := range labels {
		if err := WaitPodStatus(timeout, interval, label, "Running", finished); err != nil {
			return err
		}
	}
	return nil
}

func WaitPodsTerminated(timeout, interval time.Duration, labels ...string) error {
	for _, label := range labels {
		finished := func(output string) bool {
			return !strings.Contains(output, label)
		}
		if err := WaitPodStatus(timeout, interval, label, "terminated", finished); err != nil {
			return err
		}
	}
	return nil
}

func WaitPodStatus(timeout, interval time.Duration, label, status string, finished func(output string) bool) error {
	tick := time.Tick(interval)

	log.Debugf("waiting %v for pod %v to be %v...", timeout, label, status)
	for {
		select {
		case <-time.After(timeout):
			return fmt.Errorf("timed out waiting for %v to be %v", label, status)
		case <-tick:
			out, err := setup.KubectlOut("get", "pod", "-l", label)
			if err != nil {
				return fmt.Errorf("failed getting pod: %v", err)
			}
			if strings.Contains(out, "CrashLoopBackOff") {
				out = KubeLogs(label)
				return errors.Errorf("%v in crash loop with logs %v", label, out)
			}
			if strings.Contains(out, "ErrImagePull") || strings.Contains(out, "ImagePullBackOff") {
				out, _ = setup.KubectlOut("describe", "pod", "-l", label)
				return errors.Errorf("%v in ErrImagePull with description %v", label, out)
			}
			if finished(out) {
				return nil
			}
		}
	}
}

func KubeLogs(label string) string {
	out, err := setup.KubectlOut("logs", "-l", label)
	if err != nil {
		out = err.Error()
	}
	return out
}
