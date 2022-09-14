package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"
)

type MockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestParseRetrieveFlag(t *testing.T) {
	cases := []struct {
		name     string
		retrieve string
		want     map[string]map[string]string
		wantErr  error
	}{
		{
			name:     "empty string",
			retrieve: "",
			want:     make(map[string]map[string]string),
			wantErr:  nil,
		},
		{
			name: "happy path",
			retrieve: `
			folder1/folder2/secret1 mykey1 as key1
			folder1/folder2/secret1 mykey2 as key2
			folder1/folder2/secret2 mykey as key3
			`,
			want: map[string]map[string]string{
				"folder1/folder2/secret1": {
					"mykey1": "key1",
					"mykey2": "key2",
				},
				"folder1/folder2/secret2": {
					"mykey": "key3",
				},
			},
			wantErr: nil,
		},
		{
			name: "secret path validation",
			retrieve: `
			folder@/folder-/_secret_	mykey1 as key1
			secret$ 					mykey2 as key2
			`,
			want: nil,
			wantErr: fmt.Errorf(
				"invalid path: 'secret$'. Secret path may contain only letters, numbers, underscores, dashes, @, pluses and periods separated by colon or slash",
			),
		},
		{
			name:     "too many args",
			retrieve: `arg1 arg2 as arg3 arg4`,
			want:     nil,
			wantErr: fmt.Errorf(
				"invalid row: 'arg1 arg2 as arg3 arg4'. Expected format: '<secret path> <secret data key> as <output key>'",
			),
		},
		{
			name:     "less args",
			retrieve: `arg1 arg2`,
			want:     nil,
			wantErr: fmt.Errorf(
				"invalid row: 'arg1 arg2'. Expected format: '<secret path> <secret data key> as <output key>'",
			),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseRetrieve(tc.retrieve)
			if (tc.wantErr != nil && tc.wantErr.Error() != err.Error()) || (tc.wantErr == nil && err != nil) {
				t.Errorf("want error:\n\t%v\ngot:\n\t%v", tc.wantErr, err)
			}
			if !reflect.DeepEqual(tc.want, result) {
				t.Errorf("want:\n\t%v\ngot:\n\t%v", tc.want, result)
			}
		})
	}
}

func TestDsvGetToken(t *testing.T) {
	cases := []struct {
		name        string
		apiEndpoint string
		cid         string
		csecret     string
		client      httpClient
		want        string
		wantErr     error
	}{
		{
			name:        "happy path",
			apiEndpoint: "test.example.com",
			cid:         "client_id",
			csecret:     "client_secret",
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
			cid:         "client_id",
			csecret:     "client_secret",
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
			cid:         "client_id",
			csecret:     "client_secret",
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
			cid:         "client_id",
			csecret:     "client_secret",
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
			cid:         "client_id",
			csecret:     "client_secret",
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
			cid:         "client_id",
			csecret:     "client_secret",
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
			result, err := dsvGetToken(tc.client, tc.apiEndpoint, tc.cid, tc.csecret)
			if (tc.wantErr != nil && tc.wantErr.Error() != err.Error()) || (tc.wantErr == nil && err != nil) {
				t.Errorf("want error:\n\t%v\ngot:\n\t%v", tc.wantErr, err)
			}
			if tc.want != result {
				t.Errorf("want:\n\t%v\ngot:\n\t%v", tc.want, result)
			}
		})
	}
}

func TestDsvGetSecret(t *testing.T) {
	cases := []struct {
		name        string
		client      httpClient
		apiEndpoint string
		accessToken string
		secretPath  string
		want        map[string]interface{}
		wantErr     error
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
			secretPath:  "folder1/secret1",
			want: map[string]interface{}{
				"key": "val",
			},
			wantErr: nil,
		},
		{
			name: "bad request",
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
			secretPath:  "folder1/secret1",
			want:        nil,
			wantErr:     fmt.Errorf("API call failed: GET test.example.com/secrets/folder1/secret1: 400 Bad Request"),
		},
		{
			name: "http error",
			client: &MockHTTPClient{
				response: nil,
				err:      fmt.Errorf("error"),
			},
			apiEndpoint: "test.example.com",
			accessToken: "token",
			secretPath:  "folder1/secret1",
			want:        nil,
			wantErr:     fmt.Errorf("API call failed: error"),
		},
		{
			name: "nil body",
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
			secretPath:  "folder1/secret1",
			want:        nil,
			wantErr:     fmt.Errorf("API call failed: could not unmarshal response body: unexpected end of JSON input"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := dsvGetSecret(tc.client, tc.apiEndpoint, tc.accessToken, tc.secretPath)
			if (tc.wantErr != nil && tc.wantErr.Error() != err.Error()) || (tc.wantErr == nil && err != nil) {
				t.Errorf("want error:\n\t%v\ngot:\n\t%v", tc.wantErr, err)
			}
			if !reflect.DeepEqual(tc.want, result) {
				t.Errorf("want:\n\t%v\ngot:\n\t%v", tc.want, result)
			}
		})
	}
}

func TestOpenEnvFile(t *testing.T) {
	cases := []struct {
		name     string
		envs     map[string]string
		gitlabCI bool
		githubCI bool
		wantErr  error
	}{
		{
			name: "gitlabCI: no variable set",
			envs: map[string]string{
				"CI_JOB_NAME":     "",
				"CI_PROJECT_PATH": "",
				"GITHUB_ENV":      "",
			},
			gitlabCI: true,
			wantErr:  fmt.Errorf("CI_JOB_NAME environment is not defined"),
		},
		{
			name: "githubCI: no variable set",
			envs: map[string]string{
				"CI_JOB_NAME":     "",
				"CI_PROJECT_PATH": "",
				"GITHUB_ENV":      "",
			},
			githubCI: true,
			wantErr:  fmt.Errorf("GITHUB_ENV environment is not defined"),
		},
		{
			name: "githubCI: cannot open file",
			envs: map[string]string{
				"CI_JOB_NAME":     "",
				"CI_PROJECT_PATH": "",
				"GITHUB_ENV":      "./myfile",
			},
			githubCI: true,
			wantErr:  fmt.Errorf("cannot open file ./myfile: open ./myfile: no such file or directory"),
		},
		{
			name: "gitlabCI: no CI_PROJECT_PATH",
			envs: map[string]string{
				"CI_JOB_NAME":     "some_job",
				"CI_PROJECT_PATH": "",
				"GITHUB_ENV":      "",
			},
			gitlabCI: true,
			wantErr:  fmt.Errorf("CI_PROJECT_PATH environment is not defined"),
		},
		{
			name: "gitlabCI: cannot open file",
			envs: map[string]string{
				"CI_JOB_NAME":     "some_job",
				"CI_PROJECT_PATH": "some_project",
				"GITHUB_ENV":      "",
			},
			gitlabCI: true,
			wantErr: fmt.Errorf(
				"cannot open file /builds/some_project/some_job: open /builds/some_project/some_job: no such file or directory",
			),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			githubCI = tc.githubCI
			gitlabCI = tc.gitlabCI
			for key, val := range tc.envs {
				os.Setenv(key, val)
			}
			_, err := openEnvFile(true)
			if (tc.wantErr != nil && tc.wantErr.Error() != err.Error()) || (tc.wantErr == nil && err != nil) {
				t.Errorf("want error:\n\t%v\ngot:\n\t%v", tc.wantErr, err)
			}
		})
	}
}
