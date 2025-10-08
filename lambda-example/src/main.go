package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	rda "github.com/sil-org/rest-data-archiver"
)

type LambdaConfig struct {
	ConfigPath string
}

func main() {
	lambda.Start(handler)
}

func handler(lambdaConfig LambdaConfig) error {
	return rda.Run(lambdaConfig.ConfigPath)
}
