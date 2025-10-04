package rest_data_archiver

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sil-org/rest-data-archiver/alert"
	"github.com/sil-org/rest-data-archiver/aws"
	"github.com/sil-org/rest-data-archiver/internal"
	"github.com/sil-org/rest-data-archiver/restapi"
)

var appConfig internal.AppConfig

func Run(configFile string) error {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	log.Printf("Archive started at %s", time.Now().UTC().Format(time.RFC1123Z))

	appConfig, err := internal.LoadConfig(configFile)
	if err != nil {
		sendAlert(fmt.Sprintf("Unable to load config, error: %s", err))
		return nil
	}

	// Instantiate Source
	var source internal.Source
	switch appConfig.Source.Type {
	case internal.SourceTypeRestAPI:
		source, err = restapi.NewRestAPISource(appConfig.Source)
	default:
		err = errors.New("unrecognized source type")
	}

	if err != nil {
		sendAlert(fmt.Sprintf("Unable to initialize %s source, error: %s", appConfig.Source.Type, err))
		return nil
	}

	// Instantiate Destination
	var destination internal.Destination
	switch appConfig.Destination.Type {
	case internal.DestinationTypeS3:
		destination, err = aws.NewS3Destination(appConfig.Destination)
	default:
		err = errors.New("unrecognized destination type")
	}

	if err != nil {
		sendAlert(fmt.Sprintf("Unable to initialize %s destination, error: %s", appConfig.Destination.Type, err))
		return nil
	}

	maxNameLength := appConfig.MaxSetNameLength()
	var errors []string

	// Iterate through Sets and process changes
	for i, set := range appConfig.Sets {
		if set.Name == "" {
			msg := "configuration contains a set with no name"
			errors = append(errors, msg)
		}
		prefix := fmt.Sprintf("[ %-*s ] ", maxNameLength, set.Name)
		setLogger := log.New(os.Stdout, prefix, 0)
		setLogger.Printf("(%v/%v) Beginning archive set", i+1, len(appConfig.Sets))

		// Apply Set configs (excluding source/destination as appropriate)
		err = source.ForSet(set.Name, set.Source)
		if err != nil {
			msg := fmt.Sprintf(`Error setting source set on set "%s": %s`, set.Name, err)
			setLogger.Println(msg)
			errors = append(errors, msg)
		}

		err = destination.ForSet(set.Name, set.Destination)
		if err != nil {
			msg := fmt.Sprintf(`Error setting destination set on set "%s": %s`, set.Name, err)
			setLogger.Println(msg)
			errors = append(errors, msg)
		}

		if err := internal.RunSet(setLogger, source, destination, appConfig); err != nil {
			msg := fmt.Sprintf(`Archive failed with error on set "%s": %s`, set.Name, err)
			setLogger.Println(msg)
			errors = append(errors, msg)
		}
	}

	if len(errors) > 0 {
		sendAlert(fmt.Sprintf("Sync error(s):\n%s", strings.Join(errors, "\n")))
	}

	log.Printf("Archive completed at %s", time.Now().UTC().Format(time.RFC1123Z))
	return nil
}

func sendAlert(msg string) {
	log.Println(msg)
	alert.SendEmail(appConfig.Alert, msg)
}
