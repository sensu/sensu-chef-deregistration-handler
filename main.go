package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/go-chef/chef"
	"github.com/sensu-community/sensu-plugin-sdk/httpclient"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
)

type HandlerConfig struct {
	sensu.PluginConfig

	Endpoint      string
	ClientName    string
	ClientKeyPath string
	SSLPemPath    string
	SSLVerify     bool
	SensuAPIURL   string
	SensuAPIKey   string
	SensuCACert   string
}

type ConfigOptions struct {
	Endpoint      sensu.PluginConfigOption
	ClientName    sensu.PluginConfigOption
	ClientKeyPath sensu.PluginConfigOption
	SSLPemPath    sensu.PluginConfigOption
	SSLVerify     sensu.PluginConfigOption
	SensuAPIURL   sensu.PluginConfigOption
	SensuAPIKey   sensu.PluginConfigOption
	SensuCACert   sensu.PluginConfigOption
}

func (c *ConfigOptions) AsSlice() []*sensu.PluginConfigOption {
	return []*sensu.PluginConfigOption{
		&handlerConfigOptions.Endpoint,
		&handlerConfigOptions.ClientName,
		&handlerConfigOptions.ClientKeyPath,
		&handlerConfigOptions.SSLPemPath,
		&handlerConfigOptions.SSLVerify,
		&handlerConfigOptions.SensuAPIURL,
		&handlerConfigOptions.SensuAPIKey,
		&handlerConfigOptions.SensuCACert,
	}
}

var (
	handlerConfig = HandlerConfig{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-chef-handler",
			Short:    "A Chef keepalive handler for Sensu",
			Timeout:  10,
			Keyspace: "sensu.io/plugins/sensu-chef-handler/config",
		},
	}

	handlerConfigOptions = ConfigOptions{
		Endpoint: sensu.PluginConfigOption{
			Path:      "endpoint",
			Env:       "CHEF_ENDPOINT",
			Argument:  "endpoint",
			Shorthand: "e",
			Usage:     "The Chef Server API endpoint (URL)",
			Value:     &handlerConfig.Endpoint,
		},
		ClientName: sensu.PluginConfigOption{
			Path:      "client-name",
			Env:       "CHEF_CLIENT_NAME",
			Argument:  "client-name",
			Shorthand: "c",
			Usage:     "The Chef Client name to use when authenticating/querying the Chef Server API",
			Value:     &handlerConfig.ClientName,
		},
		ClientKeyPath: sensu.PluginConfigOption{
			Path:      "client-key-path",
			Env:       "CHEF_CLIENT_KEY_PATH",
			Argument:  "client-key-path",
			Shorthand: "k",
			Usage:     "The path to the Chef Client key to use when authenticating/querying the Chef Server API",
			Value:     &handlerConfig.ClientKeyPath,
		},
		SSLPemPath: sensu.PluginConfigOption{
			Path:      "ssl-pem-path",
			Env:       "CHEF_SSL_PEM_PATH",
			Argument:  "ssl-pem-path",
			Shorthand: "p",
			Usage:     "The Chef SSL pem file use when querying the Chef Server API",
			Value:     &handlerConfig.SSLPemPath,
		},
		SSLVerify: sensu.PluginConfigOption{
			Path:      "ssl-verify",
			Env:       "CHEF_SSL_VERIFY",
			Argument:  "ssl-verify",
			Shorthand: "s",
			Default:   true,
			Usage:     "If the SSL certificate will be verified when querying the Chef Server API",
			Value:     &handlerConfig.SSLVerify,
		},
		SensuAPIURL: sensu.PluginConfigOption{
			Path:     "sensu-api-url",
			Env:      "SENSU_API_URL",
			Argument: "sensu-api-url",
			Default:  "http://localhost:8080",
			Usage:    "The Sensu API URL",
			Value:    &handlerConfig.SensuAPIURL,
		},
		SensuAPIKey: sensu.PluginConfigOption{
			Path:     "sensu-api-key",
			Env:      "SENSU_API_KEY",
			Argument: "sensu-api-key",
			Usage:    "The Sensu API key",
			Value:    &handlerConfig.SensuAPIKey,
		},
		SensuCACert: sensu.PluginConfigOption{
			Path:     "sensu-ca-cert",
			Env:      "SENSU_CA_CERT",
			Argument: "sensu-ca-cert",
			Usage:    "The Sensu Go CA Certificate",
			Value:    &handlerConfig.SensuCACert,
		},
	}
)

func main() {
	handler := sensu.NewGoHandler(&handlerConfig.PluginConfig, handlerConfigOptions.AsSlice(), checkArgs, executeHandler)
	handler.Execute()
}

func checkArgs(event *types.Event) error {
	if event.Check.Name != "keepalive" {
		return errors.New("only keepalive events will be processed by this handler")
	}

	if len(handlerConfig.Endpoint) == 0 {
		return fmt.Errorf("--%s or %s environment variable is required",
			handlerConfigOptions.Endpoint.Argument,
			handlerConfigOptions.Endpoint.Env)
	}

	if len(handlerConfig.ClientName) == 0 {
		return fmt.Errorf("--%s or %s environment variable is required",
			handlerConfigOptions.ClientName.Argument,
			handlerConfigOptions.ClientName.Env)
	}

	if len(handlerConfig.ClientKeyPath) == 0 {
		return fmt.Errorf("--%s or %s environment variable is required",
			handlerConfigOptions.ClientKeyPath.Argument,
			handlerConfigOptions.ClientKeyPath.Env)
	}

	if len(handlerConfig.SensuAPIURL) == 0 {
		return fmt.Errorf("--%s or %s environment variable is required",
			handlerConfigOptions.SensuAPIURL.Argument,
			handlerConfigOptions.SensuAPIURL.Env)
	}

	if len(handlerConfig.SensuAPIKey) == 0 {
		return fmt.Errorf("--%s or %s environment variable is required",
			handlerConfigOptions.SensuAPIKey.Argument,
			handlerConfigOptions.SensuAPIKey.Env)
	}

	return nil
}

func executeHandler(event *types.Event) error {
	nodeName := chefNodeName(event)

	nodeExists, err := chefNodeExists(nodeName)
	if err != nil {
		return err
	}
	if nodeExists {
		fmt.Println("chef node exists:", nodeName)
		return nil
	}

	fmt.Println("chef node does not exist, removing the sensu entity")
	if err := removeSensuEntity(event); err != nil {
		return err
	}

	return nil
}

func chefNodeName(event *types.Event) string {
	return event.Entity.Name
}

func chefNodeExists(nodeName string) (bool, error) {
	keyBytes, err := ioutil.ReadFile(handlerConfig.ClientKeyPath)
	if err != nil {
		return true, fmt.Errorf("couldn't read client key from %s: %s", handlerConfig.ClientKeyPath, err)
	}

	client, err := chef.NewClient(&chef.Config{
		Name:    handlerConfig.ClientName,
		Key:     string(keyBytes),
		BaseURL: handlerConfig.Endpoint,
		SkipSSL: !handlerConfig.SSLVerify,
	})
	if err != nil {
		return true, fmt.Errorf("error setting up chef client: %s", err)
	}

	requestURL := fmt.Sprintf("%s/nodes/%s", handlerConfig.Endpoint, nodeName)

	req, err := client.NewRequest("GET", requestURL, nil)
	if err != nil {
		return true, err
	}

	res, err := client.Do(req, nil)
	if res != nil {
		defer res.Body.Close()

		if res.StatusCode == 404 {
			return false, nil
		}
	}
	if err != nil && res != nil {
		return true, fmt.Errorf("error when retrieving node from chef api: %s -- %s", res.Status, err)
	}
	if err != nil {
		return true, fmt.Errorf("error when retrieving node from chef api: %s", err)
	}

	return true, nil
}

func removeSensuEntity(event *types.Event) error {
	config := httpclient.CoreClientConfig{
		URL:    handlerConfig.SensuAPIURL,
		APIKey: handlerConfig.SensuAPIKey,
	}

	if handlerConfig.SensuCACert != "" {
		asn1Data, err := ioutil.ReadFile(handlerConfig.SensuCACert)
		if err != nil {
			return fmt.Errorf("unable to load sensu-ca-cert: %s", err)
		}
		cert, err := x509.ParseCertificate(asn1Data)
		if err != nil {
			return fmt.Errorf("invalid sensu-ca-cert: %s", err)
		}
		config.CACert = cert
	}

	client := httpclient.NewCoreClient(config)
	request, err := httpclient.NewResourceRequest("core/v2", "Entity", event.Entity.Namespace, event.Entity.Name)
	if err != nil {
		return err
	}

	if _, err := client.DeleteResource(context.Background(), request); err != nil {
		if httperr, ok := err.(httpclient.HTTPError); ok {
			if httperr.StatusCode < 500 {
				log.Printf("entity already deleted (%s/%s)", event.Entity.Namespace, event.Entity.Name)
				return nil
			}
		}
		return err
	}
	return nil
}
