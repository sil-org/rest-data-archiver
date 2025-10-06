package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/syslog"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sil-org/rest-data-archiver/internal"
)

const (
	DefaultObjectNamePrefix = "data_"
)

type S3Adapter struct {
	// DestinationConfig contains configuration common to all adapters
	DestinationConfig internal.DestinationConfig

	// S3Config contains configuration specific to this adapter
	S3Config S3Config

	// S3Set contains configuration that differs for each archive set
	S3Set S3Set
}

type S3Config struct {
	AwsConfig  Config
	BucketName string
}

type S3Set struct {
	ObjectNamePrefix string `json:"ObjectNamePrefix"`
}

func NewS3Destination(destinationConfig internal.DestinationConfig) (internal.Destination, error) {
	s, err := readConfig(destinationConfig.AdapterConfig)
	if err != nil {
		return nil, fmt.Errorf("error reading S3 destination config: %s", err)
	}

	s.DestinationConfig = destinationConfig

	return &s, nil
}

func readConfig(data []byte) (S3Adapter, error) {
	var s S3Adapter

	err := json.Unmarshal(data, &s.S3Config)
	if err != nil {
		return s, fmt.Errorf("error unmarshaling AwsConfig: %s", err)
	}

	if s.S3Config.BucketName == "" {
		return s, fmt.Errorf("config is missing an S3 bucket name")
	}
	if s.S3Config.AwsConfig.Region == "" {
		return s, fmt.Errorf("config is missing an AWS region")
	}
	if s.S3Config.AwsConfig.AccessKeyId == "" {
		return s, fmt.Errorf("config is missing an AWS access key")
	}
	if s.S3Config.AwsConfig.SecretAccessKey == "" {
		return s, fmt.Errorf("config is missing an AWS secret access key")
	}

	return s, nil
}

func (s *S3Adapter) ForSet(setName string, setConfigJson json.RawMessage) error {
	var setConfig S3Set
	err := json.Unmarshal(setConfigJson, &setConfig)
	if err != nil {
		return err
	}

	s.S3Set = setConfig

	// Defaults
	if s.S3Set.ObjectNamePrefix == "" {
		s.S3Set.ObjectNamePrefix = setName + "/" + DefaultObjectNamePrefix
	}

	return nil
}

func (s *S3Adapter) Write(data []byte, eventLog chan<- internal.EventLogItem) error {
	filename := fmt.Sprintf("%s%v", s.S3Set.ObjectNamePrefix, time.Now().UnixNano())
	if err := s.saveObject(data, filename); err != nil {
		eventLog <- internal.EventLogItem{
			Level:   syslog.LOG_ALERT,
			Message: fmt.Sprintf("error saving to S3: %s", err),
		}
		return err
	}
	eventLog <- internal.EventLogItem{
		Level:   syslog.LOG_INFO,
		Message: fmt.Sprintf("saved to %s on bucket %s", filename, s.S3Config.BucketName),
	}
	return nil
}

func (s *S3Adapter) saveObject(data []byte, fileName string) error {
	client, err := s.newS3Client()
	if err != nil {
		return fmt.Errorf("error initializing S3 client: %s", err)
	}

	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &s.S3Config.BucketName,
		Key:    &fileName,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("error saving data to %s/%s: %w", s.S3Config.BucketName, fileName, err)
	}

	return nil
}

func (s *S3Adapter) newS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(s.S3Config.AwsConfig.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s.S3Config.AwsConfig.AccessKeyId, s.S3Config.AwsConfig.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), err
}
