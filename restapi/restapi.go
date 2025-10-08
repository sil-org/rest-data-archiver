package restapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sil-org/rest-data-archiver/internal"
)

const (
	AuthTypeBasic            = "basic"
	AuthTypeBearer           = "bearer"
	AuthTypeSalesforceOauth  = "SalesforceOauth"
	DefaultBatchSize         = 10
	DefaultBatchDelaySeconds = 3
)

type RestAPI struct {
	RequestMethod     string
	BaseURL           string
	AuthType          string
	Username          string
	Password          string
	ClientID          string
	ClientSecret      string
	UserAgent         string
	BatchSize         int
	BatchDelaySeconds int
	destinationConfig internal.DestinationConfig
	setConfig         SetConfig
}

type SetConfig struct {
	Path string
}

// NewRestAPISource unmarshals the sourceConfig's ExtraJson into a RestApi struct
func NewRestAPISource(sourceConfig internal.SourceConfig) (internal.Source, error) {
	var restAPI RestAPI
	// Unmarshal ExtraJSON into RestAPI struct
	err := json.Unmarshal(sourceConfig.AdapterConfig, &restAPI)
	if err != nil {
		return &RestAPI{}, fmt.Errorf("json.Unmarshal error in adapter config:\n%s\nerror: %s",
			sourceConfig.AdapterConfig, err.Error())
	}

	restAPI.setDefaults()

	if restAPI.AuthType == AuthTypeSalesforceOauth {
		token, err := restAPI.getSalesforceOauthToken()
		if err != nil {
			log.Println(err)
			return &RestAPI{}, errors.New("error getting Oauth token: " + err.Error())
		}

		restAPI.Password = token
	}

	return &restAPI, nil
}

// ForSet sets this RestAPI structs Path value to the one in the unmarshalled setJson.
// It ensures the resulting Path attribute includes an initial "/"
func (r *RestAPI) ForSet(setName string, syncSetJson json.RawMessage) error {
	var setConfig SetConfig
	err := json.Unmarshal(syncSetJson, &setConfig)
	if err != nil {
		return fmt.Errorf("bad configuration in set '%s': %s", setName, err)
	}

	if len(setConfig.Path) == 0 {
		return errors.New("'path' is empty in sync set " + setName)
	}

	if !strings.HasPrefix(setConfig.Path, "/") {
		setConfig.Path = "/" + setConfig.Path
	}

	r.setConfig = setConfig

	return nil
}

func (r *RestAPI) Read() ([]byte, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	url := r.BaseURL + r.setConfig.Path
	request, err := r.httpRequest(r.RequestMethod, url, "", headers)
	if err != nil {
		return nil, fmt.Errorf("restAPI Read failed with http error: %s, %s, url: %s", err, request, url)
	}
	return request, nil
}

type SalesforceAuthResponse struct {
	ID          string `json:"id"`
	IssuedAt    string `json:"issued_at"`
	InstanceURL string `json:"instance_url"`
	Signature   string `json:"signature"`
	AccessToken string `json:"access_token"`
}

func (r *RestAPI) getSalesforceOauthToken() (string, error) {
	// Body params
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", r.Username)
	data.Set("password", r.Password)
	data.Set("client_id", r.ClientID)
	data.Set("client_secret", r.ClientSecret)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, r.BaseURL, strings.NewReader(data.Encode()))
	if err != nil {
		log.Println(err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading response body: %s", err.Error())
		return "", err
	}

	var authResponse SalesforceAuthResponse
	err = json.Unmarshal(bodyText, &authResponse)
	if err != nil {
		log.Printf("Unable to parse auth response, status: %v, err: %s. body: %s", resp.StatusCode, err.Error(), string(bodyText))
		return "", err
	}

	// Update BaseUrl to instance url
	r.BaseURL = strings.TrimSuffix(authResponse.InstanceURL, "/")

	return authResponse.AccessToken, nil
}

func (r *RestAPI) setDefaults() {
	if r.RequestMethod == "" {
		r.RequestMethod = http.MethodGet
	}
	if r.BatchSize <= 0 {
		r.BatchSize = DefaultBatchSize
	}
	if r.BatchDelaySeconds <= 0 {
		r.BatchDelaySeconds = DefaultBatchDelaySeconds
	}
	if r.UserAgent == "" {
		r.UserAgent = "rest-data-archiver"
	}
}

func (r *RestAPI) httpRequest(verb, url, body string, headers map[string]string) ([]byte, error) {
	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequest(verb, url, nil)
	} else {
		req, err = http.NewRequest(verb, url, strings.NewReader(body))
	}
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", r.UserAgent)

	switch r.AuthType {
	case AuthTypeBasic:
		req.SetBasicAuth(r.Username, r.Password)
	case AuthTypeBearer, AuthTypeSalesforceOauth:
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.Password))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read http response body: %s", err)
	}

	if resp.StatusCode >= 400 {
		return bodyBytes, errors.New(resp.Status)
	}

	return bodyBytes, nil
}
