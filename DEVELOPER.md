# Developer

- Devcontainer configuration included for Codespaces or [Remote Container](https://code.visualstudio.com/docs/remote/containers)

## Prerequisites For Devcontainer

- Docker
- Visual Studio Code
  - Run `code --install-extension ms-vscode-remote.remote-containers`
  - For supporting Codespaces: `code --install-extension GitHub.codespaces`

## Spin It Up

> **_NOTE_**
>
> 🐎 PERFORMANCE TIP: Using the directions provided for named container volume will optimize performance over trying to just "open in container" as there is no mounting files to your local filesystem.

Use command pallet with vscode (Control+Shift+P or F1) and type to find the command `Remote Containers: Clone Repository in Named Container`.

- Put the git clone url in.

Some extra features are included such as:

- Extensions for VSCode defined in `.devcontainers`, such as Go, Kubernetes & Docker, and some others.
- Initial placeholder `.zshrc` file included to help initialize usage of `direnv` for automatically loading default `.envrc` which contains local developement default environment variables.

### After Devcontainer Loads

1. Accept "Install Recommended Extensions" from popup, to automatically get all the preset tools, and you can choose do this without syncing so it's just for this development environment.
2. Open a new `zsh-login` terminal and allow the automatic setup to finish, as this will ensure all other required tools are setup.
   - Make sure to run `direnv allow` as it prompts you, to ensure all project and your personal environment variables (optional).
3. Run setup task:
   - Using CLI: Run `mage init`

## Troubleshooting

### Mismatch With Checksum for Go Modules

- Run `go clean -modcache && go mod tidy`.

### Connecting to Services Outside of devcontainer

You are in an isolated, self-contained Docker setup.
The ports internally aren't the same as externally in your host OS.
If the port forward isn't discovered automatically, enable it yourself, by using the port forward tab (next to the terminal tab).

1. You should see a port forward once the services are up (next to the terminal button in the bottom pane).
   1. If the click to open url doesn't work, try accessing the path manually, and ensure it is `https`.
      Example: `https://127.0.0.1:9999`

You can choose the external port to access, or even click on it in the tab and it will open in your host for you.

## Setup of GitHub Action Integration Test

Use the GitHub cli to configure the values based on the test data.
This shouldn't be sensitive stuff, just dummy values for testing retrieval.

```shell
gh secret create DSV_SECRET_PATH
gh secret create DSV_SECRET_KEY_1
gh secret create DSV_SECRET_KEY_2
gh secret create DSV_EXPECTED_VALUE_1
gh secret create DSV_EXPECTED_VALUE_2
```

For general config you'll also need:

```shell
gh secret set DSV_SERVER
gh secret set DSV_CLIENT_SECRET                                                                                          gh secret set DSV_CLIENT_ID
```

## Local Integration Testing

Ensure you've setup with `mage init` to have all the tooling.

Run `mage buildall test:integration`.

Setup a test env file to load into the dockerized test run.

```shell
# To read from local credfile (not optimal)
# DSV_CLIENT_ID=$(cat "${clientcredfile}" | jq '.clientId' --raw-output)
# DSV_CLIENT_SECRET=$(cat "${clientcredfile}" | jq '.clientSecret' --raw-output)
# INSTEAD: read using dsv cli! 💯
rm .cache/.secrets
rm .cache/.envfile
touch .cache/.envfile
cat <<EOT >> .cache/.secrets
DSV_SECRET_PATH=secrets:ci:tests:dsv-github-action:secret-01
DSV_SECRET_KEY_1=<testvaluehere>
DSV_SECRET_KEY_2=<testvaluehere>
DSV_EXPECTED_VALUE_1=<testvaluehere>
DSV_EXPECTED_VALUE_2=<testvaluehere>
DSV_DOMAIN=<tenanthere>.secretsvaultcloud.com
DSV_CLIENT_ID=$(dsv secret read "${secretpathclient}" --filter '.data.clientId')
DSV_CLIENT_SECRET=$(dsv secret read "${secretpathclient}" --filter '.data.clientSecret')
GITHUB_TOKEN=${GITHUB_TOKEN}
GITHUB_ENV=/app/.cache/.envfile
GITHUB_ACTIONS=true
DSV_SET_ENV=true
DSV_RETRIEVE=[{"secretPath": "ci:tests:dsv-github-action:secret-01", "secret_key": "", "output_variable": "RETURN_VALUE_1"}]
RUNNER_DEBUG=true
EOT
```
