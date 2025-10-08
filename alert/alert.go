package alert

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type Config struct {
	AWSRegion          string
	CharSet            string
	ReturnToAddr       string
	SubjectText        string
	RecipientEmails    []string
	AWSAccessKeyID     string
	AWSSecretAccessKey string

	sesClient *ses.Client
}

func SendEmail(c Config, body string) {
	if c.AWSAccessKeyID == "" || c.AWSSecretAccessKey == "" {
		log.Printf("AWS credentials not provided for email alerts")
		return
	}

	if err := c.setSESClient(); err != nil {
		log.Printf("error loading AWS config: %s", err)
		return
	}

	msg := types.Message{
		Subject: &types.Content{
			Charset: &c.CharSet,
			Data:    &c.SubjectText,
		},
		Body: &types.Body{
			Text: &types.Content{
				Charset: &c.CharSet,
				Data:    &body,
			},
		},
	}

	// Only report the last email error
	var lastError error
	failures := []string{}

	// Send emails to one recipient at a time to avoid one bad email sabotaging it all
	for _, to := range c.RecipientEmails {
		if err := sendEmail(msg, to, c); err != nil {
			lastError = err
			failures = append(failures, to)
		}
	}

	if lastError != nil {
		addresses := strings.Join(failures, ", ")
		log.Printf("Error sending email from '%s' to '%s': %s",
			c.ReturnToAddr, addresses, lastError.Error())
	}
}

func sendEmail(msg types.Message, to string, c Config) error {
	result, err := c.sesClient.SendEmail(context.Background(), &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Message: &msg,
		Source:  &c.ReturnToAddr,
	})
	if err != nil {
		return fmt.Errorf("error sending email, result: %+v, error: %w", result, err)
	}
	log.Printf("alert message sent to %s, message ID: %s", to, *result.MessageId)
	return nil
}

func (c *Config) setSESClient() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(c.AWSRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.AWSAccessKeyID, c.AWSSecretAccessKey, "")),
	)
	if err != nil {
		return err
	}
	c.sesClient = ses.NewFromConfig(cfg)
	return nil
}
