package services

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/RocketChat/k8s-secrets-backup/internal/options"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sService struct {
	clientset *kubernetes.Clientset
}

func NewK8sService() (*K8sService, error) {
	// #############################
	// load kubeconfig or die
	// #############################
	config := getConfigOrDie()

	// #############################
	// Create the clientset
	// #############################
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8sService{
		clientset: clientset,
	}, nil
}

func getConfigOrDie() *rest.Config {
	// #############################
	// Try to load the in-cluster config
	// #############################
	config, err := rest.InClusterConfig()
	if err == nil {
		return config
	}

	// #############################
	// Fallback to kubeconfig
	// #############################
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load Kubernetes configuration")
	}

	return config
}

func (k *K8sService) GetClusterName(opts *options.Options) (string, error) {

	// #############################
	// if the cluster name is already provided, return it
	// #############################
	if opts.ClusterInfo.Name != "" {
		log.Info().Msgf("k8s cluster name: '%s'", opts.ClusterInfo.Name)
		return opts.ClusterInfo.Name, nil
	}

	log.Info().Msgf("Cluster name not provided, trying to get it from the config map %s, key %s in the namespace %s", opts.ClusterInfo.ConfigMapName, opts.ClusterInfo.ConfigMapKey, opts.ClusterInfo.NameConfigMapNamespace)

	// #############################
	// Get the config map
	// #############################
	configMap, err := k.clientset.CoreV1().ConfigMaps(opts.ClusterInfo.NameConfigMapNamespace).Get(context.Background(), opts.ClusterInfo.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// #############################
	// Extract the cluster name from the context information
	// #############################
	clusterName, ok := configMap.Data[opts.ClusterInfo.ConfigMapKey]
	if !ok {
		return "", fmt.Errorf("cluster name not found in the '%s' field", opts.ClusterInfo.ConfigMapKey)
	}

	log.Info().Msgf("k8s cluster name: '%s'", clusterName)
	return clusterName, nil
}

func (k *K8sService) GetSecrets(fileName string, opts *options.Options) error {

	listOptions := metav1.ListOptions{}

	// #############################
	// if the secret name is provided then use it to get the secret, otherwise use the label key and value
	// #############################
	if opts.Secret.Name != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", opts.Secret.Name)
	} else {
		listOptions.LabelSelector = fmt.Sprintf("%s=%s", opts.Secret.LabelKey, opts.Secret.LabelValue)
	}

	secrets, err := k.clientset.CoreV1().Secrets(opts.Secret.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	// #############################
	// Remove resourceVersion and uid from the Secrets list
	// #############################
	secretsNames := make([]string, len(secrets.Items))

	for i, secret := range secrets.Items {
		secrets.Items[i].ResourceVersion = ""
		secrets.Items[i].UID = ""
		secrets.Items[i].ManagedFields = nil

		secretsNames[i] = secret.ObjectMeta.Name
	}

	log.Info().Msgf("Total Secrets %d, Secret Name(s): %s", len(secrets.Items), strings.Join(secretsNames, ", "))

	// #############################
	// Save the secrets into a yaml file
	// #############################
	secretList := &corev1.SecretList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "SecretList",
		},
		Items: secrets.Items,
	}

	newFile, err := os.Create(path.Join(opts.BackupDir, fileName))
	if err != nil {
		return err
	}
	defer newFile.Close()

	y := printers.YAMLPrinter{}
	err = y.PrintObj(secretList, newFile)
	if err != nil {
		return err
	}

	return nil
}
