package dga

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	env "github.com/caarlos0/env/v6"
	"github.com/pterm/pterm"
)

// defaultTimeout defines default timeout for HTTP requests.
const defaultTimeout = time.Second * 5

// PermissionReadWriteOwner is the octal permission for Read Write for the owner of the file.
const PermissionReadWriteOwner = 0o600

type Config struct {
	IsCI    bool `env:"GITHUB_ACTIONS"` // IsCI determines if the system is detecting being in CI system.
	IsDebug bool `env:"RUNNER_DEBUG"`   // IsDebug is based on github action flagging as debug/trace level.

	// DSV SPECIFIC ENV VARIABLES.

	SetEnv          bool   `env:"DSV_SET_ENV"`                         // SetEnv is only for GitHub Actions.
	DomainEnv       string `env:"DSV_DOMAIN,required"`                 // Tenant domain name (e.g. example.secretsvaultcloud.com).
	ClientIDEnv     string `env:"DSV_CLIENT_ID,required"`              // Client ID for authentication.
	ClientSecretEnv string `json:"-" env:"DSV_CLIENT_SECRET,required"` // Client Secret for authentication.
	RetrieveEnv     string `env:"DSV_RETRIEVE,required"`               // JSON formatted string with data to retrieve from DSV.
}

// RetrieveValues is the struct to put keyvalues into
//
//	type RetrieveValues struct {
//		SecretToRetrieve []SingleValue
//	}
type SecretToRetrieve struct {
	SecretPath     string `json:"secretPath"`
	SecretKey      string `json:"secretKey"`
	OutputVariable string `json:"outputVariable"`
}

// getGithubEnv reads fromt he current step target github action
// The path on the runner to the file that sets environment variables from workflow commands.
// This file is unique to the current step and changes for each step in a job.
// For example, /home/runner/work/_temp/_runner_file_commands/set_env_87406d6e-4979-4d42-98e1-3dab1f48b13a.
// For more information, see "Workflow commands for GitHub Actions.".
// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#environment-files
// The step that creates or updates the environment variable does not have access to the new value, but all subsequent steps in a job will have access.
func (cfg *Config) getGithubEnv() (string, error) {
	githubenv, isSet := os.LookupEnv("GITHUB_ENV")
	if !isSet {
		return "", fmt.Errorf("GITHUB_ENV is not set")
	}
	pterm.Debug.Printfln("GITHUB_ENV: %s", githubenv)
	pterm.Success.Printfln("getGithubEnv() success")
	return githubenv, nil
}

// configure Pterm settings for project based on the detected environment.
// Github documents their special syntax here: https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions
func (cfg *Config) configureLogging() {
	pterm.Info.Println("configureLogging()")

	pterm.Error = *pterm.Error.WithShowLineNumber().WithLineNumberOffset(1) //nolint:reassign // changing prefix later, not an issue.
	pterm.Warning = *pterm.Warning.WithShowLineNumber().WithLineNumberOffset(1)
	pterm.Warning = *pterm.Error.WithShowLineNumber().WithLineNumberOffset(1)

	pterm.Error.Prefix = pterm.Prefix{
		Text:  "::error ",
		Style: &pterm.Style{},
	}
	pterm.Debug.Prefix = pterm.Prefix{
		Text:  "::debug ",
		Style: &pterm.Style{},
	}
	pterm.Warning.Prefix = pterm.Prefix{
		Text:  "::warning ",
		Style: &pterm.Style{},
	}
	pterm.Success.Printfln("configureLogging() success")
}

func (cfg *Config) sendRequest(c HTTPClient, req *http.Request, out any) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Delinea-DSV-Client", "github-action")
	resp, err := c.Do(req)
	if err != nil {
		pterm.Error.Printfln("sendRequest: %+v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s: %s", req.Method, req.URL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		pterm.Error.Printfln("sendRequest() unable to read response body: %+v", err)
		return fmt.Errorf("could not read response body: %w", err)
	}

	if err = json.Unmarshal(body, &out); err != nil {
		pterm.Error.Printfln("Unmarshal(): %+v", err)
		return fmt.Errorf("could not unmarshal response body: %w", err)
	}
	pterm.Success.Printfln("sendRequest() success")
	return nil
}

func Run() error { //nolint:funlen,cyclop // funlen: this could use refactoring in future to break it apart more, but leaving as is at this time.
	var err error
	var retrievedValues []SecretToRetrieve

	cfg := Config{}
	cfg.configureLogging()

	if err := env.Parse(&cfg, env.Options{
		// Prefix: "DSV_",.
	}); err != nil {
		pterm.Error.Printfln("env.Parse() %+v", err)
		return fmt.Errorf("unable to parse env vars: %w", err)
	}
	pterm.Success.Println("parsed environment variables")

	actionMaskValue(cfg.ClientIDEnv)
	actionMaskValue(cfg.ClientSecretEnv)

	if cfg.IsDebug {
		pterm.Info.Println("DEBUG detected, setting debug output to enabled")
		pterm.EnableDebugMessages()
		pterm.Debug.Println("debug messages have been enabled")

		// No %v to avoid exposing secret values.
		pterm.Debug.Printfln("IsCI            : %v", cfg.IsCI)
		pterm.Debug.Printfln("IsDebug         : %v", cfg.IsDebug)

		pterm.Debug.Printfln("SetEnv          : %v", cfg.SetEnv)
		pterm.Debug.Printfln("DomainEnv       : %v", cfg.DomainEnv)
		pterm.Debug.Println("ClientIDEnv     : ** value exists, but not exposing in logs **")
		pterm.Debug.Println("ClientSecretEnv : ** value exists, but not exposing in logs **")
		pterm.Debug.Printfln("RetrieveEnv     : %v", cfg.RetrieveEnv)
	}

	retrievedValues, err = ParseRetrieve(cfg.RetrieveEnv)
	if err != nil {
		pterm.Error.Printfln("run failure: %v", err)
		return err
	}

	apiEndpoint := fmt.Sprintf("https://%s/v1", cfg.DomainEnv)
	httpClient := &http.Client{Timeout: defaultTimeout}

	token, err := DSVGetToken(httpClient, apiEndpoint, &cfg)
	if err != nil {
		pterm.Error.Printfln("authentication failure: %v", err)
		return fmt.Errorf("unable to get access token")
	}
	var envFile *os.File

	// This function will only run if both is CI and SetEnv is detected.
	if cfg.IsCI && cfg.SetEnv {
		envFile, err = ActionsOpenEnvFile(&cfg)
		if err != nil {
			pterm.Error.Printfln("unable to run actionsopenEnvFile: %v", err)
			return err
		}
		defer envFile.Close()
	}

	for _, item := range retrievedValues {
		pterm.Debug.Printfln("start processing: SecretPath: %s SecretKey: %s", item.SecretPath, item.SecretKey)
		secret, err := DSVGetSecret(httpClient, apiEndpoint, token, item, &cfg)
		if err != nil {
			pterm.Error.Printfln("%q: Failed to fetch secret: %v", item, err)
			return fmt.Errorf("unable to get secret")
		}

		secretData, ok := secret["data"].(map[string]interface{})
		if !ok {
			pterm.Error.Printfln("%q: Cannot get data from secret", item)
			return fmt.Errorf("cannot parse secret")
		}
		pterm.Success.Printfln("retrieved successfully: %q", item)

		val, ok := secretData[item.SecretKey].(string)
		if !ok {
			pterm.Error.Printfln("%q: Key %q not found in data", item, item.SecretKey)
			return fmt.Errorf("specified field was not found in data")
		}

		pterm.Debug.Printfln("%q: Found %q key in data", item, item.SecretKey)

		if !cfg.IsCI {
			continue
		}

		outputKey := item.OutputVariable
		actionSetOutput(outputKey, val) // TODO: this needs to be correctly set to use the right output variable.
		pterm.Debug.Printfln("%q: Set output %q to value in %q", item.SecretPath, outputKey, item.SecretKey)
		pterm.Success.Printfln("actionSetOutput success: %q", outputKey)

		if cfg.SetEnv {
			if err := ActionsExportVariable(envFile, outputKey, val); err != nil { // TODO: this needs to be correctly set to use the right output variable.
				pterm.Error.Printfln("%q: unable to export env variable: %v", outputKey, err)
				return fmt.Errorf("cannot set environment variable")
			}
			pterm.Success.Printfln("%q: Set env var %q to value in %q", item, strings.ToUpper(outputKey), item.SecretKey)
		}
	}
	return nil
}

func ParseRetrieve(retrieve string) ([]SecretToRetrieve, error) {
	pterm.Info.Println("parseRetrieve()")

	var retrieveThese []SecretToRetrieve
	if err := json.Unmarshal([]byte(retrieve), &retrieveThese); err != nil {
		return []SecretToRetrieve{}, fmt.Errorf("unable to unmarshal: %w", err)
	}
	pterm.Success.Printfln("parseRetrieve(): returning %+v", retrieveThese)
	return retrieveThese, nil
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func DSVGetToken(c HTTPClient, apiEndpoint string, cfg *Config) (string, error) {
	pterm.Info.Println("DSVGetToken()")
	body := []byte(fmt.Sprintf(
		`{"grant_type":"client_credentials","client_id":"%s","client_secret":"%s"}`,
		cfg.ClientIDEnv, cfg.ClientSecretEnv,
	))
	endpoint := apiEndpoint + "/token"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("could not build request: %w", err)
	}

	resp := make(map[string]interface{})
	if err = cfg.sendRequest(c, req, &resp); err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}

	token, ok := resp["accessToken"].(string)
	if !ok {
		return "", fmt.Errorf("could not read access token from response")
	}
	return token, nil
}

func DSVGetSecret(client HTTPClient, apiEndpoint, accessToken string, item SecretToRetrieve, cfg *Config) (map[string]interface{}, error) {
	pterm.Info.Println("dsvGetSecret()")
	// Endpoint := apiEndpoint + "/secrets/" + secretPath.
	endpoint, err := url.JoinPath(apiEndpoint, "secrets", item.SecretPath)
	if err != nil {
		pterm.Debug.Println("dsvGetSecret() problem with building url")
		return nil, fmt.Errorf("unable to build url: %w", err)
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		pterm.Debug.Printfln("dsvGetSecret(): endpoint: %q", endpoint)
		return nil, fmt.Errorf("could not build request: %w", err)
	}

	req.Header.Set("Authorization", accessToken)

	resp := make(map[string]interface{})
	if err = cfg.sendRequest(client, req, &resp); err != nil {
		pterm.Debug.Printfln("cfg.sendRequest() failure on sending request endpoint:%q req:%+v", endpoint, req)

		return nil, fmt.Errorf("API call failed: %w", err)
	}
	pterm.Success.Printfln("dsvGetSecret() success")
	return resp, nil
}

func actionSetOutput(key, val string) {
	fmt.Printf("::set-output name=%s::%s\n", key, val)
	actionMaskValue(val)
}

func actionMaskValue(val string) {
	fmt.Printf("::add-mask::%s\n", val)
}

// ActionsOpenEnvFile is used for writing secrets back in GitHub.
func ActionsOpenEnvFile(cfg *Config) (*os.File, error) {
	pterm.Info.Println("actionsopenEnvFile()")

	envFileName, err := cfg.getGithubEnv()
	if err != nil {
		return nil, fmt.Errorf("GITHUB_ENV environment is not defined")
	}
	_, err = os.Stat(envFileName)
	if err != nil {
		pterm.Error.Printfln("unable to validate envFileName exists: %v", err)
		return nil, fmt.Errorf("envFileName doesn't seem to exist: %w", err)
	}
	pterm.Success.Printfln("envfilepath: %s", envFileName)

	// Confirm permissions of file.
	if fi, err := os.Lstat(envFileName); err != nil {
		pterm.Warning.Println("unable to read permissions of target file")
	} else {
		pterm.Info.Printfln("envFileName permission: %#o", fi.Mode().Perm())
	}

	envFile, err := os.OpenFile(envFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, PermissionReadWriteOwner) //nolint:nosnakecase // these are standard package values and ok to leave snakecase.
	if errors.Is(err, os.ErrNotExist) {
		// See if we can provide some useful info on the existing permissions.
		return nil, fmt.Errorf("envfile doesn't exist or has denied permission %s: %w", envFileName, err)
	}
	if err != nil {
		return nil, fmt.Errorf("general error cannot open file %s: %w", envFileName, err)
	}
	pterm.Success.Printfln("actionsopenEnvFile() success")
	return envFile, nil
}

func ActionsExportVariable(envFile *os.File, key, val string) error {
	pterm.Info.Println("actionsExportVariable()")
	if _, err := envFile.WriteString(fmt.Sprintf("%s=%s\n", strings.ToUpper(key), val)); err != nil {
		return fmt.Errorf("could not update %s environment file: %w", envFile.Name(), err)
	}
	pterm.Success.Printfln("actionsExportVariable() success")
	return nil
}
