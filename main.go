package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/pterm/pterm"
)

// List of environment variables names used as input.
const ()

// defaultTimeout defines default timeout for HTTP requests.
const defaultTimeout = time.Second * 5

// PermissionReadWriteOwner is the octal permission for Read Write for the owner of the file.
const PermissionReadWriteOwner = 0o600

type config struct {
	IsCI    bool `env:"GITHUB_ACTION"` // IsCI determines if the system is detecting being in CI system.
	isDebug bool `env:"RUNNER_DEBUG"`  // IsDebug is based on github action flagging as debug/trace level.
	SetEnv  bool `env:"SET_ENV"`       // SetEnv is only for GitHub Actions.

	DomainEnv       string `env:"DOMAIN,required"`        // Tenant domain name (e.g. example.secretsvaultcloud.com).
	ClientIDEnv     string `env:"CLIENT_ID,required"`     // Client ID for authentication.
	ClientSecretEnv string `env:"CLIENT_SECRET,required"` // Client Secret for authentication.
	RetrieveEnv     string `env:"RETRIEVE,required"`      // Rows with data to retrieve from DSV in format `<path> <data key> as <output key>`.
}

// getGithubEnv reads from the current step target github action
// The path on the runner to the file that sets environment variables from workflow commands.
// This file is unique to the current step and changes for each step in a job.
// For example, /home/runner/work/_temp/_runner_file_commands/set_env_87406d6e-4979-4d42-98e1-3dab1f48b13a.
// For more information, see "Workflow commands for GitHub Actions.".
func (cfg *config) getGithubEnv() (string, error) {
	githubenv, isSet := os.LookupEnv("GITHUB_ENV")
	if !isSet {
		return "", fmt.Errorf("GITHUB_ENV is not set")
	}
	return githubenv, nil
}

func (cfg *config) sendRequest(c httpClient, req *http.Request, out any) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Delinea-DSV-Client", "github-action")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s: %s", req.Method, req.URL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %w", err)
	}

	if err = json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("could not unmarshal response body: %w", err)
	}
	return nil
}

// configure Pterm settings for project based on the detected environment.
func (cfg *config) configureLogging() {
	pterm.Info.Println("configureLogging()")
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "::error::",
		Style: &pterm.Style{},
	}
	pterm.Debug.Prefix = pterm.Prefix{
		Text:  "::debug::",
		Style: &pterm.Style{},
	}
	if cfg.isDebug {
		pterm.EnableDebugMessages()
		pterm.Debug.Println("debug messages have been enabled")
	}
}

func main() {
	cfg := &config{}
	if err := env.Parse(&cfg); err != nil {
		pterm.Error.Printfln("%+v", err)
	}

	cfg.configureLogging()

	pterm.Debug.Printfln("%+v", cfg)

	if err := run(cfg.DomainEnv, cfg.ClientIDEnv, cfg.ClientSecretEnv, cfg.RetrieveEnv, cfg); err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
}

func run(domain, clientID, clientSecret, retrieve string, cfg *config) error {
	var err error
	retrieveData, err := parseRetrieve(retrieve)
	if err != nil {
		return err
	}

	apiEndpoint := fmt.Sprintf("https://%s/v1", domain)
	httpClient := &http.Client{Timeout: defaultTimeout}

	pterm.Info.Println("🔑 Fetching access token")
	token, err := dsvGetToken(httpClient, apiEndpoint, clientID, clientSecret, cfg)
	if err != nil {
		pterm.Debug.Printfln("Authentication failed: %v", err)
		return fmt.Errorf("unable to get access token")
	}
	var envFile *os.File

	// This function will only run if both is CI and SetEnv is detected.
	if cfg.IsCI && cfg.SetEnv {
		envFile, err = actionsopenEnvFile(cfg)
		if err != nil {
			return err
		}
		defer envFile.Close()
	}

	pterm.Info.Println("✨ Fetching secret(s) from DSV")

	for path, dataMap := range retrieveData {
		pterm.Debug.Printfln("%q: Start processing", path)

		secret, err := dsvGetSecret(httpClient, apiEndpoint, token, path, cfg)
		if err != nil {
			pterm.Debug.Printfln("%q: Failed to fetch secret: %v", path, err)
			return fmt.Errorf("unable to get secret")
		}

		secretData, ok := secret["data"].(map[string]interface{})
		if !ok {
			pterm.Debug.Printfln("%q: Cannot get data from secret", path)
			return fmt.Errorf("cannot parse secret")
		}

		for dataKey, outputKey := range dataMap {
			val, ok := secretData[dataKey].(string)
			if !ok {
				pterm.Debug.Printfln("%q: Key %q not found in data", path, dataKey)
				return fmt.Errorf("specified field was not found in data")
			}
			pterm.Debug.Printfln("%q: Found %q key in data", path, dataKey)

			if cfg.IsCI {
				actionSetOutput(outputKey, val)
				pterm.Debug.Printfln("%q: Set output %q to value in %q", path, outputKey, dataKey)
			}
			if cfg.IsCI && cfg.SetEnv {
				if err := actionsExportVariable(envFile, outputKey, val); err != nil {
					pterm.Debug.Printfln("%q: Exporting variable error: %v", path, err)
					return fmt.Errorf("cannot set environment variable")
				}
				pterm.Debug.Printfln("%q: Set env var %q to value in %q", path, strings.ToUpper(outputKey), dataKey)
			}
		}
	}
	return nil
}

func parseRetrieve(retrieve string) (map[string]map[string]string, error) {
	pathRegexp := regexp.MustCompile(`^[a-zA-Z0-9:\/@\+._-]+$`)
	whitespaces := regexp.MustCompile(`\s+`)

	result := make(map[string]map[string]string)

	for _, row := range strings.Split(retrieve, "\n") {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		row = whitespaces.ReplaceAllString(row, " ")

		tokens := strings.Split(row, " ")
		if len(tokens) != 4 { //nolint:gomnd // ok to list 4 as this is the parsed token just for this function
			return nil, fmt.Errorf(
				"invalid row: '%s'. Expected format: '<secret path> <secret data key> as <output key>'", row,
			)
		}

		var (
			path      = tokens[0]
			dataKey   = tokens[1]
			outputKey = tokens[3]
		)
		if !pathRegexp.MatchString(path) {
			return nil, fmt.Errorf(
				"invalid path: '%s'. Secret path may contain only letters, numbers, underscores, dashes, @, pluses and periods separated by colon or slash",
				path,
			)
		}

		if _, ok := result[path]; !ok {
			result[path] = make(map[string]string)
		}
		result[path][dataKey] = outputKey
	}

	return result, nil
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func dsvGetToken(c httpClient, apiEndpoint, cid, csecret string, cfg *config) (string, error) {
	body := []byte(fmt.Sprintf(
		`{"grant_type":"client_credentials","client_id":"%s","client_secret":"%s"}`,
		cid, csecret,
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

func dsvGetSecret(c httpClient, apiEndpoint, accessToken, secretPath string, cfg *config) (map[string]interface{}, error) {
	endpoint := apiEndpoint + "/secrets/" + secretPath
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("could not build request: %w", err)
	}

	req.Header.Set("Authorization", accessToken)

	resp := make(map[string]interface{})
	if err = cfg.sendRequest(c, req, &resp); err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	return resp, nil
}

func actionSetOutput(key, val string) {
	fmt.Printf("::set-output name=%s::%s\n", key, val)
}

// actionsopenEnvFile is used for writing secrets back in GitHub.
func actionsopenEnvFile(cfg *config) (*os.File, error) {
	var (
		envFileName string
		envFile     *os.File
		err         error
	)

	envFileName, err = cfg.getGithubEnv()
	if err != nil {
		return nil, fmt.Errorf("GITHUB_ENV environment is not defined")
	}

	envFile, err = os.OpenFile(envFileName, os.O_APPEND|os.O_WRONLY, PermissionReadWriteOwner)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %w", envFileName, err)
	}
	return envFile, nil
}

func actionsExportVariable(envFile *os.File, key, val string) error {
	if _, err := envFile.WriteString(fmt.Sprintf("%s=%s\n", strings.ToUpper(key), val)); err != nil {
		return fmt.Errorf("could not update %s environment file: %w", envFile.Name(), err)
	}
	return nil
}
