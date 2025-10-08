package internal

import (
	"encoding/json"
	"log/syslog"

	"github.com/sil-org/rest-data-archiver/alert"
)

type SourceConfig struct {
	Type          string
	AdapterConfig json.RawMessage
}

type DestinationConfig struct {
	Type          string
	AdapterConfig json.RawMessage
}

type RuntimeConfig struct {
	DryRunMode bool
}

type AppConfig struct {
	Runtime     RuntimeConfig
	Source      SourceConfig
	Destination DestinationConfig
	Alert       alert.Config
	Sets        []Set
}

type Set struct {
	Name        string
	Source      json.RawMessage
	Destination json.RawMessage
}

type EventLogItem struct {
	Message string
	Level   syslog.Priority
}

func (l EventLogItem) String() string {
	return LogLevels[l.Level] + ": " + l.Message
}

var LogLevels = map[syslog.Priority]string{
	syslog.LOG_EMERG:   "Emerg",
	syslog.LOG_ALERT:   "Alert",
	syslog.LOG_CRIT:    "Critical",
	syslog.LOG_ERR:     "Error",
	syslog.LOG_WARNING: "Warning",
	syslog.LOG_NOTICE:  "Notice",
	syslog.LOG_INFO:    "Info",
	syslog.LOG_DEBUG:   "Debug",
}

type Destination interface {
	ForSet(setName string, setJson json.RawMessage) error
	Write(data []byte, activityLog chan<- EventLogItem) error
}

type Source interface {
	ForSet(setName string, setJson json.RawMessage) error
	Read() ([]byte, error)
}
