package internal

import (
	"encoding/json"
	"errors"
	"log"
	"log/syslog"
	"os"
	"time"

	"github.com/sil-org/rest-data-archiver/alert"
)

const (
	DefaultConfigFile = "./config.json"
	DefaultVerbosity  = 5
	DestinationTypeS3 = "S3"
	SourceTypeRestAPI = "RestAPI"
)

// LoadConfig looks for a config file if one is provided. Otherwise, it looks for
// a config file based on the CONFIG_PATH env var.  If that is not set, it gets
// the default config file ("./config.json").
func LoadConfig(configFile string) (AppConfig, error) {
	if configFile == "" {
		configFile = os.Getenv("CONFIG_PATH")
		if configFile == "" {
			configFile = DefaultConfigFile
		}
	}

	log.Printf("Using config file: %s\n", configFile)

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("unable to read application config file %s, error: %s\n", configFile, err.Error())
		return AppConfig{}, err
	}

	config, err := parseConfig(data)
	if err != nil {
		return AppConfig{}, err
	}

	return config, nil
}

func parseConfig(data []byte) (AppConfig, error) {
	config := AppConfig{}
	err := json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("unable to unmarshal application configuration file data, error: %s\n", err.Error())
		return config, err
	}

	if config.Source.Type == "" {
		return config, errors.New("configuration appears to be missing a Source configuration")
	}

	if config.Destination.Type == "" {
		return config, errors.New("configuration appears to be missing a Destination configuration")
	}

	log.Printf("Configuration loaded. Source type: %s, Destination type: %s\n", config.Source.Type, config.Destination.Type)
	log.Printf("%v Archive sets found:\n", len(config.Sets))

	for i, set := range config.Sets {
		log.Printf("  %v) %s\n", i+1, set.Name)
	}
	return config, nil
}

// RunSet calls the source API and writes the result to the destination adapter
func RunSet(logger *log.Logger, source Source, destination Destination, config AppConfig) error {
	sourceData, err := source.Read()
	if err != nil {
		return err
	}

	// If in DryRun mode only print out the config and any results from calling the source API
	if config.Runtime.DryRunMode {
		logger.Println("Dry-run mode enabled. No data will be written to the destination.")
		printSourceResponse(logger, sourceData)
		return nil
	}

	// Create a channel to pass activity logs for printing
	eventLog := make(chan EventLogItem, 50)
	go processEventLog(logger, config.Alert, eventLog)

	if err := destination.Write(sourceData, eventLog); err != nil {
		logger.Println("Error saving to destination:", err.Error())
	} else {
		logger.Println("Data saved to destination")
	}

	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 10)
		if len(eventLog) == 0 {
			break
		}
	}
	close(eventLog)

	return nil
}

func processEventLog(logger *log.Logger, config alert.Config, eventLog <-chan EventLogItem) {
	for msg := range eventLog {
		logger.Println(msg)
		if msg.Level == syslog.LOG_ALERT || msg.Level == syslog.LOG_EMERG {
			alert.SendEmail(config, msg.String())
		}
	}
}

func printSourceResponse(logger *log.Logger, response []byte) {
	if len(response) > 500 {
		logger.Printf("response:\n%s...(truncated)", response[0:500])
	} else {
		logger.Printf("response:\n%s\n", response)
	}
}

type EmptyDestination struct{}

func (e *EmptyDestination) ForSet(setJson json.RawMessage) error {
	return nil
}

type EmptySource struct{}

func (e *EmptySource) ForSet(setJson json.RawMessage) error {
	return nil
}

func (a *AppConfig) MaxSetNameLength() int {
	maxLength := 0
	for _, set := range a.Sets {
		if maxLength < len(set.Name) {
			maxLength = len(set.Name)
		}
	}
	return maxLength
}
