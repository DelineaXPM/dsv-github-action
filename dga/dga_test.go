package dga_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
	"github.com/pterm/pterm"

	dga "github.com/DelineaXPM/dsv-github-action/dga"
)

type MockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestParseRetrieveFlag(t *testing.T) {
	pterm.DisableOutput()
	cases := []struct {
		name     string
		retrieve string
		want     []dga.SecretToRetrieve
		wantErr  error
	}{
		{
			name:     "empty string",
			retrieve: "",
			want:     []dga.SecretToRetrieve{},
			wantErr:  nil,
		},
		{
			name: "happy path",
			retrieve: `
			[
				{"secretPath": "folder1/folder2/secret1", "secretKey": "mykey1", "outputVariable": ""},
				{"secretPath": "folder1/folder2/secret1", "secretKey": "mykey2", "outputVariable": ""},
				{"secretPath": "folder1/folder2/secret2", "secretKey": "key3", "outputVariable": ""}
			]
			`,
			want: []dga.SecretToRetrieve{
				{
					SecretPath:     "folder1/folder2/secret1",
					SecretKey:      "mykey1",
					OutputVariable: "",
				},
				{
					SecretPath:     "folder1/folder2/secret1",
					SecretKey:      "mykey2",
					OutputVariable: "",
				},
				{
					SecretPath:     "folder1/folder2/secret2",
					SecretKey:      "key3",
					OutputVariable: "",
				},
			},
			wantErr: nil,
		},
		{
			name: "invalid json input structure",
			retrieve: `
			[
				{"arg1": "path", "arg2": "path", "arg3": ""},
			]
			`,
			want: nil,
			wantErr: fmt.Errorf(
				"invalid json structure. Expected format: '[{\"secretPath\": \"path\",\"secretKey\": \"data key lookup here\", \"outputVariable\": \"ACTIONS_ENV_VAR\"}]' ",
			),
		},
	}
	is := is.New(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t) //nolint:govet // shadow is fine, chill

			result, err := dga.ParseRetrieve(tc.retrieve)

			if tc.wantErr != nil {
				is.True(err != nil) // Should produce error.
			} else {
				is.Equal(tc.want, result) // Result should match desired []dga.SecretToRetrieve.
			}
		})
	}
}

func TestDsvGetToken(t *testing.T) {
	pterm.DisableOutput()
	is := is.New(t)

	cfg := &dga.Config{
		IsCI:            true,
		ClientIDEnv:     "client_id",
		ClientSecretEnv: "client_secret",
	}
	cases := []struct {
		name        string
		apiEndpoint string
		cid         string
		csecret     string
		client      dga.HTTPClient
		want        string
		wantErr     error
	}{
		{
			name:        "happy path",
			apiEndpoint: "test.example.com",

			client: &MockHTTPClient{
				response: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`{
						"accessToken": "token"
					}`))),
				},
				err: nil,
			},
			want:    "token",
			wantErr: nil,
		},
		{
			name:        "bad request",
			apiEndpoint: "test.example.com",

			client: &MockHTTPClient{
				response: &http.Response{
					Status:     "400 Bad Request",
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte(nil))),
				},
				err: nil,
			},
			want:    "",
			wantErr: fmt.Errorf("API call failed: POST test.example.com/token: 400 Bad Request"),
		},
		{
			name:        "empty endpoint",
			apiEndpoint: "",

			client: &MockHTTPClient{
				response: &http.Response{
					Status:     "400 Bad Request",
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte(nil))),
				},
				err: nil,
			},
			want:    "",
			wantErr: fmt.Errorf("API call failed: POST /token: 400 Bad Request"),
		},
		{
			name:        "http error",
			apiEndpoint: "test.example.com",

			client: &MockHTTPClient{
				response: nil,
				err:      fmt.Errorf("error"),
			},
			want:    "",
			wantErr: fmt.Errorf("API call failed: error"),
		},
		{
			name:        "nil body",
			apiEndpoint: "test.example.com",

			client: &MockHTTPClient{
				response: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(nil))),
				},
				err: nil,
			},
			want:    "",
			wantErr: fmt.Errorf("API call failed: could not unmarshal response body: unexpected end of JSON input"),
		},
		{
			name:        "no access token",
			apiEndpoint: "test.example.com",

			client: &MockHTTPClient{
				response: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`{
						"test": "token"
					}`))),
				},
				err: nil,
			},
			want:    "",
			wantErr: fmt.Errorf("could not read access token from response"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := dga.DSVGetToken(tc.client, tc.apiEndpoint, cfg)

			if (tc.wantErr != nil && tc.wantErr.Error() != err.Error()) || (tc.wantErr == nil && err != nil) {
				// T.Errorf("want error:\n\t%v\ngot:\n\t%v", tc.wantErr, err).
				is.Equal(tc.wantErr, err) // Error should match.
			}
			is.Equal(tc.want, result) // Result should match.
		})
	}
}

func TestDsvGetSecret(t *testing.T) {
	pterm.DisableOutput()
	cfg := &dga.Config{
		IsCI: true,
	}
	cases := []struct {
		name           string
		client         dga.HTTPClient
		apiEndpoint    string
		accessToken    string
		itemToRetrieve dga.SecretToRetrieve
		want           map[string]interface{}
		wantErr        bool
	}{
		{
			name: "happy path",
			client: &MockHTTPClient{
				response: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`{
						"key": "val"
					}`))),
				},
				err: nil,
			},
			apiEndpoint: "test.example.com",
			accessToken: "token",
			itemToRetrieve: dga.SecretToRetrieve{
				SecretPath: "folder1/secret1",
				SecretKey:  "key",
			},
			want: map[string]interface{}{
				"key": "val",
			},
			wantErr: false,
		},
		{
			name: "GET secret should fail with 400",
			client: &MockHTTPClient{
				response: &http.Response{
					Status:     "400 Bad Request",
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte(nil))),
				},
				err: nil,
			},
			apiEndpoint: "test.example.com",
			accessToken: "token",
			itemToRetrieve: dga.SecretToRetrieve{
				SecretPath: "folder1/secret1",
				SecretKey:  "key",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "GET should fail",
			client: &MockHTTPClient{
				response: nil,
				err:      fmt.Errorf("error"),
			},
			apiEndpoint: "test.example.com",
			accessToken: "token",
			itemToRetrieve: dga.SecretToRetrieve{
				SecretPath: "folder1/secret1",
				SecretKey:  "key",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nil body should fail on unmarshaling",
			client: &MockHTTPClient{
				response: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(nil))),
				},
				err: nil,
			},
			apiEndpoint: "test.example.com",
			accessToken: "token",
			itemToRetrieve: dga.SecretToRetrieve{
				SecretPath: "folder1/secret1",
				SecretKey:  "key",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		is := is.New(t)
		t.Run(tc.name, func(t *testing.T) {
			result, err := dga.DSVGetSecret(tc.client, tc.apiEndpoint, tc.accessToken, tc.itemToRetrieve, cfg)
			if tc.wantErr {
				is.True(err != nil) // Should fail due to file missing.
			} else {
				is.Equal(tc.want, result) // Returned result should match expected value.
			}
		})
	}
}

func TestOpenEnvFile(t *testing.T) {
	// TODO: https://github.com/spf13/afero 👉🏻 look at abstraction so we can easily test filesystem calls.
	pterm.EnableDebugMessages()
	pterm.DisableOutput()
	is := is.New(t)
	const cacheDir = ".cache"
	const everyoneReadWriteExec = 0o777
	is.NoErr(os.MkdirAll(cacheDir, everyoneReadWriteExec))
	validSecretFile := filepath.Join(cacheDir, ".valid-secret-file")
	invalidSecretFile := filepath.Join(cacheDir, ".invalid-secret-file")
	unreadableSecretFile := filepath.Join(cacheDir, ".unreadable-secret-file")
	// _, err := os.Create(validSecretFile)
	// is.NoErr(err) // Creating a local file should succeed.
	// Trunk-ignore(golangci-lint/gosec).

	is.NoErr(os.WriteFile(validSecretFile, []byte("foo=bar"), everyoneReadWriteExec))      // Writing to a local file with unreadable set should succeed.
	is.NoErr(os.WriteFile(unreadableSecretFile, []byte("foo=bar"), everyoneReadWriteExec)) // Writing to a local file should succeed.
	is.NoErr(os.Chmod(unreadableSecretFile, os.FileMode(0o111)))                           // Should set permissions to write only.

	defer func() {
		is.NoErr(os.Chmod(unreadableSecretFile, dga.PermissionReadWriteOwner)) // Should reset cachedir permissions to default.
		is.NoErr(os.RemoveAll(cacheDir))                                       // Should remove cachedir.
	}()

	cases := []struct {
		name       string
		envs       map[string]string
		isCI       bool
		shouldFail bool
	}{
		{
			name: "githubCI should fail when required variables",
			envs: map[string]string{
				"CI_JOB_NAME":     "",
				"CI_PROJECT_PATH": "",
				"GITHUB_ENV":      "",
			},
			isCI:       true,
			shouldFail: true,
		},
		{
			name: "githubCI should fail when unable to open envfile",
			envs: map[string]string{
				"CI_JOB_NAME":     "",
				"CI_PROJECT_PATH": "",
				"GITHUB_ENV":      invalidSecretFile,
			},
			isCI:       true,
			shouldFail: true,
		},
		{
			name: "githubCI should fail when file is not readable",
			envs: map[string]string{
				"CI_JOB_NAME":     "",
				"CI_PROJECT_PATH": "",
				"GITHUB_ENV":      unreadableSecretFile,
			},
			isCI:       true,
			shouldFail: true,
		},
		{
			name: "githubCI should succeed when file exists and is writeable",
			envs: map[string]string{
				"CI_JOB_NAME":     "",
				"CI_PROJECT_PATH": "",
				"GITHUB_ENV":      validSecretFile,
			},
			isCI:       true,
			shouldFail: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t) //nolint:govet // ok to shadow for subtest

			cfg := &dga.Config{
				IsCI: tc.isCI,
			}
			for key, val := range tc.envs {
				os.Setenv(key, val)
			}
			_, err := dga.ActionsOpenEnvFile(cfg)
			if tc.shouldFail {
				is.True(err != nil) // Should fail.
			} else {
				is.NoErr(err) // Should open envfile without issue.
			}
		})
	}
}
