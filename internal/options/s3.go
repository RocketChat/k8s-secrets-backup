package options

import "errors"

type S3 struct {
	BucketName   string `env:"BUCKET_NAME"`
	Path         string `env:"PATH"`
	Region       string `env:"REGION"`
	Endpoint     string `env:"ENDPOINT"`
	AccessKey    string `env:"ACCESS_KEY"`
	SecretKey    string `env:"SECRET_KEY"`
	UsePathStyle bool   `env:"USE_PATH_STYLE"`
}

func (s3 *S3) Validate() error {
	// #############################
	// provide the bucket name
	// #############################
	if s3.BucketName == "" {
		return errors.New("S3__BUCKET_NAME is required")
	}

	// #############################
	// provide the region or endpoint
	// #############################
	if s3.Region == "" {
		return errors.New("provide either S3__REGION")
	}

	// #############################
	// provide the access key
	// #############################
	if s3.AccessKey == "" {
		return errors.New("S3__ACCESS_KEY is required")
	}

	// #############################
	// provide the secret key
	// #############################
	if s3.SecretKey == "" {
		return errors.New("S3__SECRET_KEY is required")
	}
	return nil
}
