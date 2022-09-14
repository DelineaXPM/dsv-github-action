# Repo

<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->

[![All Contributors](https://img.shields.io/badge/all_contributors-3-orange.svg?style=flat-square)](#contributors-)

<!-- ALL-CONTRIBUTORS-BADGE:END -->

Brief description

## Getting Started

- [Developer](DEVELOPER.md): instructions on running tests, local tooling, and other resources.
- [DSV Documentation](https://docs.delinea.com/dsv/current?ref=githubrepo)

## How This Works

## Inputs

| Name           | Description                                                              |
| -------------- | ------------------------------------------------------------------------ |
| `domain`       | Tenant domain name (e.g. example.secretsvaultcloud.com).                 |
| `clientId`     | Client ID for authentication.                                            |
| `clientSecret` | Client Secret for authentication.                                        |
| `setEnv`       | Set environment variables. Applicable only for GitHub Actions.           |
| `retrieve`     | Data to retrieve from DSV in format `<path> <data key> as <output key>`. |

## Prerequisites

This plugin uses authentication based on Client Credentials, i.e. via Client ID and Client Secret.

You can generate Client Credentials using a command-line interface (CLI) tool. Latest version of
the CLI tool can be found here: <https://dsv.secretsvaultcloud.com/downloads>. Quick start with
the CLI: <https://docs.delinea.com/dsv/current/quickstart>.

To create a role run:

```shell
dsv role create --name <role name>
```

To generate a pair of Client ID and Client Secret run:

```shell
dsv client create --role <role name>
```

Use returned values of Client ID and Client Secret to configure this plugin. After this you can
create secrets for the pipeline and configure access to those secrets.

Example of configuration:

```shell
# Create a role named "ci-reader":
dsv role create --name ci-reader

# Generate client credentials for the role:
dsv client create --role ci-reader

# Create a secret:
dsv secret create \
  --path 'ci-secrets:secret1' \
  --data '{"password":"foo","token":"bar"}'

# Create a policy to allow role "ci-reader" to read secrets under "ci-secrets":
dsv policy create \
  --path 'secrets:ci-secrets' \
  --actions 'read' \
  --effect 'allow' \
  --subjects 'roles:ci-reader'
```

## GitHub usage example

See [integration.yaml](.github/workflows/integration.yaml) for an example of how to use this to retrieve secrets and use outputs on other tasks.

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
