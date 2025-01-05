package options

import "fmt"

type ClusterInfo struct {
	Name                   string `env:"NAME"`
	NameConfigMapNamespace string `env:"NAME_CONFIG_MAP_NAMESPACE"`
	ConfigMapName          string `env:"NAME_CONFIG_MAP_NAME"`
	ConfigMapKey           string `env:"NAME_CONFIG_MAP_KEY"`
}

func (k *ClusterInfo) Validate() error {
	// ################################
	// provide either CLUSTER__NAME or CLUSTER__NAMESPACE, CLUSTER__NAME_CONFIG_MAP_NAME, and CLUSTER__NAME_CONFIG_MAP_KEY
	// ################################
	if (k.Name == "" && (k.NameConfigMapNamespace == "" || k.ConfigMapName == "" || k.ConfigMapKey == "")) ||
		(k.Name != "" && (k.NameConfigMapNamespace != "" || k.ConfigMapName != "" || k.ConfigMapKey != "")) {
		return fmt.Errorf("provide either CLUSTER__NAME or CLUSTER__NAMESPACE, CLUSTER__NAME_CONFIG_MAP_NAME, and CLUSTER__NAME_CONFIG_MAP_KEY")
	}

	return nil
}
