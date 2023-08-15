package main

import (
	"os"
)

var (
	// secret resources filters
	secretName string
	namespace  string
	labelKey   string
	labelValue string

	// s3
	bucketName string
	s3folder   string
	s3region   string

	// aws creds
	accessKeyID     string
	secretAccessKey string

	// age
	ageRecipientPublicKey string
)

func init() {
	// secret resources filters
	secretName = os.Getenv("SECRET_NAME")

	namespace = os.Getenv("NAMESPACE")
	if namespace == "" {
		panic("please provide the environment variable NAMESPACE")
	}

	labelKey = os.Getenv("LABEL_KEY")
	labelValue = os.Getenv("LABEL_VALUE")

	if secretName == "" {
		if labelKey == "" || labelValue = "" {
			panic("please provide either the environmental variable SECRET_NAME or both environmental variables LABEL_KEY and LABEL_VALUE")
		}
	} else {
		if labelKey != "" || labelValue != "" {
			panic("please provide either the environmental variable SECRET_NAME or both environmental variables LABEL_KEY and LABEL_VALUE")
		}
	}

	// s3
	bucketName = os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		panic("please provide the environment variable BUCKET_NAME")
	}

	s3folder = os.Getenv("S3_FOLDER")
	if s3folder == "" {
		panic("please provide the environment variable S3_FOLDER")
	}

	s3region = os.Getenv("S3_REGION")
	if s3region == "" {
		panic("please provide the environment variable S3_REGION")
	}

	// aws creds
	accessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKeyID == "" {
		panic("please provide the environment variable AWS_ACCESS_KEY_ID")
	}

	secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		panic("please provide the environment variable AWS_SECRET_ACCESS_KEY")
	}

	// age
	ageRecipientPublicKey = os.Getenv("AGE_PUBLIC_KEY")
	if ageRecipientPublicKey == "" {
		panic("please provide the environment variable AGE_PUBLIC_KEY")
	}
}
