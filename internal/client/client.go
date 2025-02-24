package client

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/onedr0p/exportarr/internal/model"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Client struct is a Radarr client to request an instance of a Radarr
type Client struct {
	config     *cli.Context
	configFile *model.Config
	httpClient http.Client
}

// NewClient method initializes a new Radarr client.
func NewClient(c *cli.Context, cf *model.Config) *Client {
	return &Client{
		config:     c,
		configFile: cf,
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// DoRequest - Take a HTTP Request and return Unmarshaled data
func (c *Client) DoRequest(endpoint string, target interface{}) error {
	apiVersion := "v3"

	if c.config.Command.Name == "lidarr" {
		apiVersion = "v1"
	}

	var url string
	var apiKey string

	// Use the values from config.xml if using the config flag
	if c.config.String("config") != "" {
		url = fmt.Sprintf("%s:%s/%s/api/%s/%s",
			c.config.String("url"),
			c.configFile.Port,
			c.configFile.UrlBase,
			apiVersion,
			endpoint,
		)
		apiKey = c.configFile.ApiKey
	} else {
		// Otherwise use the value provided in the api-key flag
		url = fmt.Sprintf("%s/api/%s/%s",
			c.config.String("url"),
			apiVersion,
			endpoint,
		)
		apiKey = c.config.String("api-key")
	}

	log.Infof("Sending HTTP request to %s", url)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: c.config.Bool("disable-ssl-verify")}
	req, err := http.NewRequest("GET", url, nil)
	if c.config.String("basic-auth-username") != "" && c.config.String("basic-auth-password") != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s",
			base64.StdEncoding.EncodeToString([]byte(c.config.String("basic-auth-username")+":"+c.config.String("basic-auth-password"))),
		))
	}
	req.Header.Add("X-Api-Key", apiKey)

	if err != nil {
		log.Fatalf("An error has occurred when creating HTTP request %v", err)
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatalf("An error has occurred during retrieving statistics %v", err)
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		errMsg := fmt.Sprintf("An error has occurred during retrieving statistics HTTP statuscode %d", resp.StatusCode)
		log.Fatal(errMsg)
		return errors.New(errMsg)
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}
