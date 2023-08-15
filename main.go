package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/cli-runtime/pkg/printers"

	"filippo.io/age"
	"filippo.io/age/armor"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {

	log.Printf("secret-name: %s\n namespace: %s\n label-key: %s\n label-value: %s\n bucket-name: %s\n s3-folder: %s\n s3-region: %s\n",
		secretName, namespace, labelKey, labelValue, bucketName, s3folder, s3region)

	// creates the in-cluster config
	config, err := rest.InClusterConfig()

	if err != nil {
		log.Fatal(err.Error())
	}

	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Get k8s cluster name
	clusterName, err := getClusterName(clientset)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Set not encrypted file name with secret(s)
	var baseFileName string
	if secretName != "" {
		baseFileName = fmt.Sprintf("%s-%s.yaml", clusterName, secretName)
	} else {
		baseFileName = fmt.Sprintf("%s-%s-%s.yaml", clusterName, labelKey, labelValue)
		baseFileName = strings.ReplaceAll(baseFileName, "/", "_")
	}

	// Get the current time and set desire format
	currentTime := time.Now().UTC()
	timeStamp := currentTime.Format("2006-01-02_15-04-05") // YYYY-MM-DD_HH-MM-SS

	fileName := fmt.Sprintf("%s-%s", baseFileName, timeStamp)
	encryptedFileName := fileName + ".age.asc"
	s3key := fmt.Sprintf("%s%s", s3folder, encryptedFileName)

	log.Println("not encrypted secrets file name:", fileName)
	log.Println("encrypted secrets file name:", encryptedFileName)
	log.Println("s3 key:", s3key)

	// Get secrets to backup
	err = saveSecretsIntoYaml(clientset, secretName, namespace, labelKey, labelValue, fileName)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("Saving secrets into yaml file succesfully")

	// Encrypt the secrets backup file
	err = encryptSecretsFile(ageRecipientPublicKey, fileName, encryptedFileName)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("File '%s' encrypted to '%s'\n", fileName, encryptedFileName)

	// Upload to backup s3 bucket the encrypted file
	err = uploadFileToS3(accessKeyID, secretAccessKey, bucketName, s3key, encryptedFileName, s3region)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("File uploaded successfully!")

}

func getClusterName(clientset *kubernetes.Clientset) (string, error) {
	ctx := context.TODO()

	// Get the current context's information
	currentContext, err := clientset.CoreV1().ConfigMaps("kube-system").Get(ctx, "cluster-info", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Extract the cluster name from the context information
	clusterName, ok := currentContext.Data["cluster-name"]
	if !ok {
		return "", fmt.Errorf("cluster name not found in the cluster-name field")
	}

	log.Printf("k8s cluster name: '%s'\n", clusterName)
	return clusterName, nil
}

func saveSecretsIntoYaml(clientset *kubernetes.Clientset, secretName,
	namespace, labelKey, labelValue, fileName string) error {

	listOptions := metav1.ListOptions{}
	if labelKey != "" && labelValue != "" {
		listOptions.LabelSelector = fmt.Sprintf("%s=%s", labelKey, labelValue)
	}
	if secretName != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", secretName)
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), listOptions)

	if err != nil {
		log.Fatal(err)
	}

	// Remove resourceVersion and uid from the Secrets list
	for i := range secrets.Items {
		secrets.Items[i].ResourceVersion = ""
		secrets.Items[i].UID = ""
	}

	for _, secret := range secrets.Items {
		log.Printf("Secret Name: %s\n", secret.ObjectMeta.Name)
	}

	log.Printf("Total Secrets: %d\n", len(secrets.Items))

	secretList := &corev1.SecretList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "SecretList",
		},
		Items: secrets.Items,
	}

	// Save the secret list in a yaml file
	newFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err.Error())
	}
	y := printers.YAMLPrinter{}
	defer newFile.Close()

	err = y.PrintObj(secretList, newFile)
	if err != nil {
		return fmt.Errorf("unable save secrets in local file: %w", err)
	}

	return nil

}

func encryptSecretsFile(ageRecipientPublicKey, fileName, encryptedFile string) error {

	// Open the input file for reading
	in, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	// Create the output file for writing the encrypted content
	out, err := os.Create(encryptedFile)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// Create an Age encryption writer
	recipient, err := age.ParseX25519Recipient(ageRecipientPublicKey)
	if err != nil {
		log.Fatal(err)
	}

	// Encrypt the input file with ASCII armor
	aw := armor.NewWriter(out)
	defer aw.Close()

	encWriter, err := age.Encrypt(aw, recipient)
	if err != nil {
		log.Fatal(err)
	}
	defer encWriter.Close()

	// Copy the contents of the input file to the encryption writer
	_, err = io.Copy(encWriter, in)
	if err != nil {
		log.Fatalf("unable to encrypt secrets file: %v", err)
	}

	return nil

}

func uploadFileToS3(accessKeyID, secretAccessKey, bucketName, s3key, encryptedFileName, s3region string) error {

	creds := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")

	// Load AWS config using the custom credentials provider
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(s3region),
		config.WithCredentialsProvider(creds),
	)

	if err != nil {
		return fmt.Errorf("unable to load aws SDK config: %w", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg)

	// Open the file for reading
	file, err := os.Open(encryptedFileName)
	if err != nil {
		return fmt.Errorf("unable to open file for uploading to s3: %w", err)
	}
	defer file.Close()

	// Upload the file to S3
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("unable to upload file to S3: %w", err)
	}

	return nil
}
