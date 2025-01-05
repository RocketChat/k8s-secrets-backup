package options

import "errors"

type K8sSecret struct {
	Name       string `env:"NAME"`
	Namespace  string `env:"NAMESPACE"`
	LabelKey   string `env:"LABEL_KEY"`
	LabelValue string `env:"LABEL_VALUE"`
}

func (k8s *K8sSecret) Validate() error {
	// #############################
	// provide either the secret name or the label key and label value
	// #############################
	if (k8s.Name != "" && (k8s.LabelKey != "" || k8s.LabelValue != "")) ||
		(k8s.Name == "" && (k8s.LabelKey == "" || k8s.LabelValue == "")) {
		return errors.New("provide either SECRET__NAME or both  SECRET__LABEL_KEY and SECRET__LABEL_VALUE")
	}

	// #############################
	// provide the namespace
	// #############################
	if k8s.Namespace == "" {
		return errors.New("SECRET__NAMESPACE is required")
	}

	return nil
}
