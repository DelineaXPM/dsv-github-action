package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

// List of environment variables names used as input.
const (
	DomainEnv       = "DOMAIN"        // Tenant domain name (e.g. example.secretsvaultcloud.com).
	ClientIDEnv     = "CLIENT_ID"     // Client ID for authentication.
	ClientSecretEnv = "CLIENT_SECRET" // Client Secret for authentication.
	RetrieveEnv     = "RETRIEVE"      // Rows with data to retrieve from DSV in format `<path> <data key> as <output key>`.
	SetEnvEnv       = "SET_ENV"       // Set env variables. Applicable only for GitHub Actions.
)

// defaultTimeout defines default timeout for HTTP requests.
const defaultTimeout = time.Second * 5

// PermissionReadWriteOwner is the octal permission for Read Write for the owner of the file.
const PermissionReadWriteOwner = 0o600

var (
	githubCI      = os.Getenv("GITHUB_ACTION") != ""
	gitlabCI      = os.Getenv("GITLAB_CI") != ""
	gitlabCIDebug = os.Getenv("GITLAB_CI_DEBUG") != ""
)

func main() {
	switch {
	case githubCI:
		info("🐣 Start working with GitHub CI.")
	case gitlabCI:
		info("🐣 Start working with GitLab CI.")
	default:
		printError(fmt.Errorf("🤡 Unknown CI server"))
		os.Exit(1)
	}

	readEnv := func(name string) string {
		val := os.Getenv(name)
		if val == "" {
			printError(fmt.Errorf("environment variable %q is required and cannot be empty", name))
			os.Exit(1)
		}
		return val
	}

	domain := readEnv(DomainEnv)
	clientID := readEnv(ClientIDEnv)
	clientSecret := readEnv(ClientSecretEnv)
	retrieve := readEnv(RetrieveEnv)
	setEnv := (githubCI && os.Getenv(SetEnvEnv) != "") || gitlabCI

	if err := run(domain, clientID, clientSecret, retrieve, setEnv); err != nil {
		printError(err)
		os.Exit(1)
	}
}

func run(domain, clientID, clientSecret, retrieve string, setEnv bool) error {
	retrieveData, err := parseRetrieve(retrieve)
	if err != nil {
		return err
	}

	apiEndpoint := fmt.Sprintf("https://%s/v1", domain)
	httpClient := &http.Client{Timeout: defaultTimeout}

	info("🔑 Fetching access token...")
	token, err := dsvGetToken(httpClient, apiEndpoint, clientID, clientSecret)
	if err != nil {
		debugf("Authentication failed: %v", err)
		return fmt.Errorf("unable to get access token")
	}

	envFile, err := openEnvFile(setEnv)
	if err != nil {
		return err
	}
	defer envFile.Close()

	info("✨ Fetching secret(s) from DSV...")

	for path, dataMap := range retrieveData {
		debugf("%q: Start processing...", path)

		secret, err := dsvGetSecret(httpClient, apiEndpoint, token, path)
		if err != nil {
			debugf("%q: Failed to fetch secret: %v.", path, err)
			return fmt.Errorf("unable to get secret")
		}

		secretData, ok := secret["data"].(map[string]interface{})
		if !ok {
			debugf("%q: Cannot get data from secret.", path)
			return fmt.Errorf("cannot parse secret")
		}

		for dataKey, outputKey := range dataMap {
			val, ok := secretData[dataKey].(string)
			if !ok {
				debugf("%q: Key %q not found in data.", path, dataKey)
				return fmt.Errorf("specified field was not found in data")
			}
			debugf("%q: Found %q key in data.", path, dataKey)

			if githubCI {
				actionSetOutput(outputKey, val)
				debugf("%q: Set output %q to value in %q.", path, outputKey, dataKey)
			}
			if setEnv {
				if err := exportVariable(envFile, outputKey, val); err != nil {
					debugf("%q: Exporting variable error: %v.", path, err)
					return fmt.Errorf("cannot set environment variable")
				}
				debugf("%q: Set env var %q to value in %q.", path, strings.ToUpper(outputKey), dataKey)
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

func dsvGetToken(c httpClient, apiEndpoint, cid, csecret string) (string, error) {
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
	if err = sendRequest(c, req, &resp); err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}

	token, ok := resp["accessToken"].(string)
	if !ok {
		return "", fmt.Errorf("could not read access token from response")
	}
	return token, nil
}

func dsvGetSecret(c httpClient, apiEndpoint, accessToken, secretPath string) (map[string]interface{}, error) {
	endpoint := apiEndpoint + "/secrets/" + secretPath
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("could not build request: %w", err)
	}

	req.Header.Set("Authorization", accessToken)

	resp := make(map[string]interface{})
	if err = sendRequest(c, req, &resp); err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	return resp, nil
}

func sendRequest(c httpClient, req *http.Request, out any) error {
	req.Header.Set("Content-Type", "application/json")
	if githubCI {
		req.Header.Set("Delinea-DSV-Client", "github-action")
	} else if gitlabCI {
		req.Header.Set("Delinea-DSV-Client", "gitlab-job")
	}

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

func debug(s string) {
	if githubCI {
		fmt.Printf("::debug::%s\n", s)
	} else if gitlabCI && gitlabCIDebug {
		fmt.Printf("##[debug]\x1b[94m%s\x1b[0m\n", s)
	}
}

func debugf(format string, args ...interface{}) {
	debug(fmt.Sprintf(format, args...))
}

func info(s string) {
	if githubCI {
		fmt.Println(s)
	} else if gitlabCI {
		fmt.Printf("\x1b[92m%s\x1b[0m\n", s)
	}
}

func printError(err error) {
	if githubCI {
		fmt.Printf("::error::%v\n", err)
	} else if gitlabCI {
		fmt.Printf("\x1b[91m%v\x1b[0m\n", err)
	}
}

func actionSetOutput(key, val string) {
	fmt.Printf("::set-output name=%s::%s\n", key, val)
}

func openEnvFile(setEnv bool) (*os.File, error) {
	var (
		envFile *os.File
		err     error
	)
	if gitlabCI {
		jobName := os.Getenv("CI_JOB_NAME")
		if jobName == "" {
			return nil, fmt.Errorf("CI_JOB_NAME environment is not defined")
		}
		pwd := os.Getenv("CI_PROJECT_PATH")
		if pwd == "" {
			return nil, fmt.Errorf("CI_PROJECT_PATH environment is not defined")
		}
		envFileName := path.Join("/builds/", pwd, jobName)
		envFile, err = os.OpenFile(envFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, PermissionReadWriteOwner)
		if err != nil {
			return nil, fmt.Errorf("cannot open file %s: %w", envFileName, err)
		}
	} else if githubCI && setEnv {
		envFileName := os.Getenv("GITHUB_ENV")
		if envFileName == "" {
			return nil, fmt.Errorf("GITHUB_ENV environment is not defined")
		}
		envFile, err = os.OpenFile(envFileName, os.O_APPEND|os.O_WRONLY, PermissionReadWriteOwner)
		if err != nil {
			return nil, fmt.Errorf("cannot open file %s: %w", envFileName, err)
		}
	}
	return envFile, nil
}

func exportVariable(envFile *os.File, key, val string) error {
	if _, err := envFile.WriteString(fmt.Sprintf("%s=%s\n", strings.ToUpper(key), val)); err != nil {
		return fmt.Errorf("could not update %s environment file: %w", envFile.Name(), err)
	}
	return nil
}
