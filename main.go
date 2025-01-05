package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-envconfig"

	"filippo.io/age"
	"filippo.io/age/armor"

	"github.com/RocketChat/k8s-secrets-backup/internal/options"
	"github.com/RocketChat/k8s-secrets-backup/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	// #############################
	// Load environment variables
	// #############################
	godotenv.Load()

	// #############################
	// prepare the options
	// #############################
	var opts options.Options
	if err := envconfig.Process(context.Background(), &opts); err != nil {
		log.Fatal().Err(err).Msg("Failed to process environment variables")
	}

	// #############################
	// validate the options
	// #############################
	if err := opts.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// #############################
	// prepare k8s service
	// #############################
	k8sService, err := services.NewK8sService()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create k8s service")
	}

	// #############################
	// Get the cluster name
	// #############################
	clusterName, err := k8sService.GetClusterName(&opts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get cluster name")
	}

	// #############################
	// Prepare the file name, s3 key, and encrypted file name
	// #############################
	var baseFileName string
	if opts.Secret.Name == "" {
		baseFileName = fmt.Sprintf("%s-%s-%s", clusterName, opts.Secret.LabelKey, opts.Secret.LabelValue)
		baseFileName = strings.ReplaceAll(baseFileName, "/", "_")
	} else {
		baseFileName = fmt.Sprintf("%s-%s", clusterName, opts.Secret.Name)
	}

	timeStamp := time.Now().UTC().Format("2006-01-02_15-04-05") // YYYY-MM-DD_HH-MM-SS
	fileName := fmt.Sprintf("%s-%s.yaml", baseFileName, timeStamp)
	encryptedFileName := fileName + ".age.asc"
	s3key := path.Join(opts.S3.Path, encryptedFileName)

	log.Info().Msgf("not encrypted secrets file name: %s", fileName)
	log.Info().Msgf("encrypted secrets file name: %s", encryptedFileName)
	log.Info().Msgf("s3 key: %s", s3key)

	// #############################
	// Get secrets to backup
	// #############################
	if err = k8sService.GetSecrets(fileName, &opts); err != nil {
		log.Fatal().Err(err).Msg("Failed to save secrets into yaml file")
	}

	// Encrypt the secrets backup file
	if err := encryptSecretsFile(opts, fileName, encryptedFileName); err != nil {
		log.Fatal().Err(err).Msg("Failed to encrypt secrets file")
	}

	log.Info().Msgf("File '%s' encrypted to '%s'", fileName, encryptedFileName)

	// #############################
	// prepare s3 service
	// #############################
	s3Service, err := services.NewS3Service(&opts.S3)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create s3 service")
	}

	// #############################
	// Upload to backup s3 bucket the encrypted file
	// #############################
	if err = s3Service.UploadFile(&opts, s3key, encryptedFileName); err != nil {
		log.Fatal().Err(err).Msg("Failed to upload file to S3")
	}

	log.Info().Msgf("File uploaded successfully!")
}

func encryptSecretsFile(opts options.Options, fileName string, encryptedFile string) error {

	// #############################
	// Open the input file for reading
	// #############################
	in, err := os.Open(path.Join(opts.BackupDir, fileName))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open file for reading")
	}
	defer in.Close()

	// #############################
	// Create the output file for writing the encrypted content
	// #############################
	out, err := os.Create(path.Join(opts.BackupDir, encryptedFile))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create file for writing")
	}
	defer out.Close()

	// #############################
	// Create an Age encryption writer
	// #############################
	recipient, err := age.ParseX25519Recipient(opts.AgePublicKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse recipient public key")
	}

	// #############################
	// Encrypt the input file with ASCII armor
	// #############################
	aw := armor.NewWriter(out)
	defer aw.Close()

	encWriter, err := age.Encrypt(aw, recipient)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create encryption writer")
	}
	defer encWriter.Close()

	// #############################
	// Copy the contents of the input file to the encryption writer
	// #############################
	_, err = io.Copy(encWriter, in)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to encrypt secrets file")
	}

	return nil
}
