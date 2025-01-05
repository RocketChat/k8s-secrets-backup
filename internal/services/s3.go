package services

import (
	"context"
	"os"
	"path"

	"github.com/RocketChat/k8s-secrets-backup/internal/options"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	Client *s3.Client
}

func NewS3Service(opts *options.S3) (*S3Service, error) {

	// #############################
	// prepare configuration
	// #############################
	creds := credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(opts.Region),
		config.WithCredentialsProvider(creds),
	)

	if err != nil {
		return nil, err
	}

	// #############################
	// Create an S3 client
	// #############################
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if opts.Endpoint != "" {
			o.BaseEndpoint = aws.String(opts.Endpoint)
		}

		if opts.UsePathStyle {
			o.UsePathStyle = true
		}
	})

	return &S3Service{Client: client}, nil
}

func (s *S3Service) UploadFile(opts *options.Options, s3Key string, encryptedFileName string) error {

	// #############################
	// Open the file for reading
	// #############################
	file, err := os.Open(path.Join(opts.BackupDir, encryptedFileName))
	if err != nil {
		return err
	}
	defer file.Close()

	// #############################
	// Upload the file to S3
	// #############################
	_, err = s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(opts.S3.BucketName),
		Key:    aws.String(s3Key),
		Body:   file,
	})

	if err != nil {
		return err
	}

	return nil
}
