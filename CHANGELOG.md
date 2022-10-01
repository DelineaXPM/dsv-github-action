# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.0.6] - 2022-09-30

- abb94bc ci: release permissions refined
- e605bc7 chore: fix invalid syntax in release-composite
- 3853b81 chore: apply lint fixes for better dockerfile in devcontainer && changelog type use github
- d7c9f77 ci: minor refactoring and debugging on release automation
- 8536707 ci(release-composite): inherit secrets in attempt to resolve failing composite release
- a0d56f0 ci: make lint workflow optional and not a blocker for release
- 260b838 ci: assign specific composite action permissions
- 3c5509c style: apply trunk format fixes
- 03ac324 ci: concurrency is now scoped to the workflow itself
- 8374d37 ci: remove trigger on test for any push
- ab2ef9e ci: remove triggering integration & test on normal commit to main
- 00ecd70 ci(integration): wait for concurrent jobs to complete before proceeding
- 1861f28 ci(release-composite): support reuse of workflows with workflow_call
- 503d4b6 ci(goreleaser): add changelog generation as github type
- 05f692a ci(release-composite): initial draft of a multistage release

## [v1.0.5] - 2022-09-29

- d220c50 docs: add gg-delinea as a contributor for userTesting (#19)
- 9547220 docs: improve setup docs with minor syntax improvements (#18)
- 862b6d3 docs: minor adjustments (#17)
- ab67f2d ci: GITHUB_TOKEN is not required (#16)

## [v1.0.4] - 2022-09-28

- 2b2a341 chore: add missing metadata for name of github action

## [v1.0.3] - 2022-09-28

- 4704656 ci(goreleaser): improve changelog generation

## [v1.0.2] - 2022-09-28

- 6b8064b docs: align action name to repo for clarity
- 5a7d2d7 fix: do not iterate over items in secret data (#15)

## [v1.0.1] - 2022-09-27

- e048f49 docs: fix yaml schema to provide icon under branding
- 36f7a83 docs: fix incorrect yaml description for formatting
- 42a2aeb docs: improve github action details
- 9805df8 docs: shorten description
- 6a8e95a chore(vendor): add vendor dir & fix mage task call
- 3e1a843 ci(actions/release): use mage invoked release instead of goreleaser
- 2432bd6 chore(mage): move syft to ci stage setup
- b7bfb89 ci: add workflow dispatch to release
- 5e5cf3c ci: add goreleaser release action (#14)
- 7962a42 docs: improve docs for onboarding and usage (#13)
- c29c310 chore(deps): update actions/stale action to v6 (#10)
- 823b831 chore(deps): update amannn/action-semantic-pull-request digest to 505e44b (#11)
- 63f6d35 feat: 🎉 initial release of dsv-github-action
- 68e9b14 Initial commit
