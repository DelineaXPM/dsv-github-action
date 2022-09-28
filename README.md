# dsv-github-action

<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->

[![All Contributors](https://img.shields.io/badge/all_contributors-3-orange.svg?style=flat-square)](#contributors-)

<!-- ALL-CONTRIBUTORS-BADGE:END -->

Use Delinea DevOps Secret Vault for retrieval of your secrets.

Now, instead of storing all your secrets directly in your GitHub repo configuration, store client credentials to connect and retrieve the desired secret or multiple secrets from your secure vault.

## Getting Started

- [Developer](DEVELOPER.md): instructions on running tests, local tooling, and other resources.
- [DSV Documentation](https://docs.delinea.com/dsv/current?ref=githubrepo)
- [Download DSV CLI](https://dsv.secretsvaultcloud.com/downloads)
  Quick install example (adjust to platform/version): `curl -fSsl https://dsv.secretsvaultcloud.com/downloads/cli/1.37.5/dsv-darwin-x64 -o dsv && chmod +x ./dsv && sudo mv ./dsv /usr/local/bin`
- Remaining readme for the usage directions.
- Install [github-cli](https://cli.github.com/) for easier setup.
  - quick: `brew install gh` or see [installation instructions](https://github.com/cli/cli#installation)

## How This Works

## Inputs

| Name           | Description                                                    |
| -------------- | -------------------------------------------------------------- |
| `domain`       | Tenant domain name (e.g. example.secretsvaultcloud.com).       |
| `clientId`     | Client ID for authentication.                                  |
| `clientSecret` | Client Secret for authentication.                              |
| `setEnv`       | Set environment variables. Applicable only for GitHub Actions. |
| `retrieve`     | Data to retrieve from DSV in json format.                      |

## Prerequisites

This plugin uses authentication based on Client Credentials, i.e. via Client ID and Client Secret.

```shell
rolename="github-dsv-github-action-tests"
secretpath="ci:tests:dsv-github-action"
secretpathclient="clients:${secretpath}"

desc="a secret for testing operation of secrets against dsv-github-action"
clientcredfile=".cache/${rolename}.json"
clientcredname="${rolename}"

dsv role create --name "${rolename}"

# Option 1: Less Optimal - Save Credential to local json for testing
# dsv client create --role "${rolename}" --out "file:${clientcredfile}"

# Option 2: 🔒 MOST SECURE
# Create credential info for dsv, and set as variable. Then use the github cli to set as a secret for your action.
# Create an org secret instead if you want to share this credential in many repos.

# compress to a single line
clientcred=$(dsv client create --role "${rolename}" --plain | jq -c)

# configure the dsv server, such as mytenant.secretsvaultcloud.com
gh secret set DSV_CLIENT_ID

# use the generated client credentials in your repo
gh secret set DSV_CLIENT_ID --body "$( echo "${clientcred}" | jq '.clientId' )"
gh secret set DSV_CLIENT_SECRET --body "$( echo "${clientcred}" | jq '.clientSecret')"
```

For further setup, here's how you could create extend that script block above with also creating a secret and the policy to read just this secret.

```shell
# Create a secret
secretkey="secret-01"
secretvalue='{"value1":"taco","value2":"burrito"}'
dsv secret create \
  --path "secrets:${secretpath}:${secretkey}" \
  --data "${secretvalue}" \
  --desc "${desc}"

# Create a policy to allow role "$rolename" to read secrets under "ci:tests:integration-configs/dsv-github-action":
dsv policy create \
  --path "secrets:${secretpath}" \
  --actions 'read' \
  --effect 'allow' \
  --subjects "roles:$rolename" \
  --desc "${desc}" \
  --resources "${secretpath}:<.*>"
```

## GitHub usage example

See [integration.yaml](.github/workflows/integration.yaml) for an example of how to use this to retrieve secrets and use outputs on other tasks.

## Other Usage Examples

### Retrieve A Single Secret Without Setting Env File

```yaml
jobs:
  integration:
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - uses: actions/checkout@v3
      - id: dsv
        uses: delineaxpm/dsv-github-action@v1 # renovate: tag=v1
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          domain: ${{ secrets.DSV_SERVER }}
          clientId: ${{ secrets.DSV_CLIENT_ID }}
          clientSecret: ${{ secrets.DSV_CLIENT_SECRET }}
          setEnv: false
          retrieve: |
            [
             {"secretPath": "ci:tests:dsv-github-action:secret-01", "secretKey": "value1" }
            ]
```

### Set Output to Job Scope Using SetEnv

```yaml
jobs:
  integration:
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - uses: actions/checkout@v3
      - id: dsv
        uses: delineaxpm/dsv-github-action@v1 # renovate: tag=v1
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          domain: ${{ secrets.DSV_SERVER }}
          clientId: ${{ secrets.DSV_CLIENT_ID }}
          clientSecret: ${{ secrets.DSV_CLIENT_SECRET }}
          setEnv: true
          retrieve: |
            [
             {"secretPath": "ci:tests:dsv-github-action:secret-01", "secretKey": "value1", "outputVariable": "MY_ENV_VAR" }
            ]
      - name: validate-first-value
        if: always()
        run: |
          "This is a secret value you shouldn't echo 👉 ${{ steps.dsv.outputs.MY_ENV_VAR }}"
```

### Retrieve 2 Values from Same Secret

The json expects an array, so just add a new line.

```yaml
retrieve: |
  [
   {"secretPath": "ci:tests:dsv-github-action:secret-01", "secretKey": "value1", "outputVariable": "MY_ENV_VAR_1" },
   {"secretPath": "ci:tests:dsv-github-action:secret-01", "secretKey": "value2", "outputVariable": "MY_ENV_VAR_2" }
  ]
```

### Retrieve 2 Values from Different Secrets

> Note: Make sure your generated client credentials are associated a policy that has rights to read the different secrets.

```yaml
retrieve: |
  [
   {"secretPath": "ci:tests:dsv-github-action:secret-01", "secretKey": "value1", "outputVariable": "MY_ENV_VAR_1" },
   {"secretPath": "ci:tests:dsv-github-action:secret-02", "secretKey": "value1", "outputVariable": "MY_ENV_VAR_2" }
  ]
```

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center"><a href="https://github.com/mariiatuzovska"><img src="https://avatars.githubusercontent.com/u/41679258?v=4?s=100" width="100px;" alt=""/><br /><sub><b>Mariia</b></sub></a><br /><a href="https://github.com/DelineaXPM/dsv-github-action/commits?author=mariiatuzovska" title="Code">💻</a></td>
      <td align="center"><a href="https://www.sheldonhull.com/"><img src="https://avatars.githubusercontent.com/u/3526320?v=4?s=100" width="100px;" alt=""/><br /><sub><b>sheldonhull</b></sub></a><br /><a href="https://github.com/DelineaXPM/dsv-github-action/commits?author=sheldonhull" title="Code">💻</a></td>
      <td align="center"><a href="https://github.com/andrii-zakurenyi"><img src="https://avatars.githubusercontent.com/u/85106843?v=4?s=100" width="100px;" alt=""/><br /><sub><b>andrii-zakurenyi</b></sub></a><br /><a href="https://github.com/DelineaXPM/dsv-github-action/commits?author=andrii-zakurenyi" title="Code">💻</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!
