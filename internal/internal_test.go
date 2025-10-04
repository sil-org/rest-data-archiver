package internal

import (
	"io"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// disable log output
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func Test_parseConfig(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    AppConfig
		wantErr string
	}{
		{
			name:    "empty file",
			data:    []byte(""),
			want:    AppConfig{},
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "missing Source",
			data:    []byte(`{"Destination":{"Type":"S3"}}`),
			want:    AppConfig{},
			wantErr: "missing a Source configuration",
		},
		{
			name:    "missing Destination",
			data:    []byte(`{"Source":{"Type":"RestAPI"}}`),
			want:    AppConfig{},
			wantErr: "missing a Destination configuration",
		},
		{
			name: "minimal",
			data: []byte(`{"Source":{"Type":"RestAPI"},"Destination":{"Type":"S3"}}`),
			want: AppConfig{
				Source:      SourceConfig{Type: SourceTypeRestAPI},
				Destination: DestinationConfig{Type: DestinationTypeS3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConfig(tt.data)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr, "parseConfig() incorrect error")
				return
			}
			require.Equal(t, tt.want, got, "parseConfig() got = %v, want %v", got, tt.want)
		})
	}
}
