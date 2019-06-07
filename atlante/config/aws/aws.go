package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/go-spatial/tegola/dict"
	"os"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	// EnvAWSRegion aws region environment variable
	EnvAWSRegion = "AWS_REGION"

	// EnvAWSEndPoint aws end point environment variable
	EnvAWSEndPoint = "AWS_ENDPOINT"

	// DefaultRegion is the default aws region
	DefaultRegion = "us-east-1"

	// DefaultAccessKey for aws
	DefaultAccessKey = ""

	// DefaultSecretKey for aws
	DefaultSecretKey = ""

	// DefaultEndPoint for aws, but allows for use with other aws compatible s3 stores
	DefaultEndPoint = ""

	// ConfigKeyRegion is the aws region [optional] defaults to "us-east-1"
	ConfigKeyRegion = "region"

	// ConfigKeyEndPoint is the aws end point to hit [optional] defaults to ""
	ConfigKeyEndPoint = "end_point"

	// ConfigKeyAWSAccessKeyID the aws access key [optional] defaults to ""
	ConfigKeyAWSAccessKeyID = "aws_access_key_id"

	// ConfigKeyAWSSecretKey the aws secret key [optional] defaults to ""
	ConfigKeyAWSSecretKey = "aws_secret_access_key"

)

func NewSession(cfg dict.Dicter) (sess *session.Session, err error){
	// check for region env var
	region := os.Getenv(EnvAWSRegion)
	if region == "" {
		region = DefaultRegion
	}
	region, err = cfg.String(ConfigKeyRegion, &region)
	if err != nil {
		return nil, err
	}

	// check for endpoint env var
	endpoint := os.Getenv(EnvAWSEndPoint)
	if endpoint == "" {
		endpoint = DefaultEndPoint
	}
	endpoint, err = cfg.String(ConfigKeyEndPoint, &endpoint)
	if err != nil {
		return nil, err
	}

	accessKey := DefaultAccessKey
	accessKey, err = cfg.String(ConfigKeyAWSAccessKeyID, &accessKey)
	if err != nil {
		return nil, err
	}

	secretKey := DefaultSecretKey
	secretKey, err = cfg.String(ConfigKeyAWSSecretKey, &secretKey)
	if err != nil {
		return nil, err
	}

	awsConfig := aws.Config{
		Region: aws.String(region),
	}

	// support for static credentials, this is not recommended by AWS but
	// necessary for some enviroments

	if accessKey != "" && secretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
	}

	// if an endpoint is set, add it to the awsConfig
	// otherwise do not set it and it will automatically use the correct aws-s3 endpoint
	if endpoint != "" {
		awsConfig.Endpoint = aws.String(endpoint)
	}

	return session.NewSession(&awsConfig)
}