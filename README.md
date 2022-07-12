# cloudscale-metrics-collector

[![Build](https://img.shields.io/github/workflow/status/vshn/cloudscale-metrics-collector/Test)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/cloudscale-metrics-collector)
[![Version](https://img.shields.io/github/v/release/vshn/cloudscale-metrics-collector)][releases]
[![Maintainability](https://img.shields.io/codeclimate/maintainability/vshn/cloudscale-metrics-collector)][codeclimate]
[![Coverage](https://img.shields.io/codeclimate/coverage/vshn/cloudscale-metrics-collector)][codeclimate]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/cloudscale-metrics-collector/total)][releases]

[build]: https://github.com/vshn/cloudscale-metrics-collector/actions?query=workflow%3ATest
[releases]: https://github.com/vshn/cloudscale-metrics-collector/releases
[codeclimate]: https://codeclimate.com/github/vshn/cloudscale-metrics-collector

Template repository for common Go setups

## Features

* GitHub Workflows
  - Build (Go & Docker image)
  - Test (including CodeClimate)
  - Lint (Go)
  - Release (Goreleaser & Changelog generator)

* GitHub issue templates
  - PR template
  - Issue templates using GitHub issue forms

* Goreleaser
  - Go build for `amd64`, `armv8`
  - Docker build for `latest` and `vx.y.z` tags
  - Push Docker image to GitHub's registry `ghcr.io`

* CLI and logging framework
  - To help get you started with CLI subcommands, flags and environment variables
  - If you don't need subcommands, remove `example_command.go` and adjust `cli.App` settings in `main.go`


## Other repository settings

1. GitHub Settings
   - "Options > Wiki" (disable)
   - "Options > Allow auto-merge" (enable)
   - "Options > Automatically delete head branches" (enable)
   - "Collaborators & Teams > Add Teams and users to grant maintainer permissions
   - "Branches > Branch protection rules":
     - Branch name pattern: `master`
     - Require status check to pass before merging: `["lint"]` (you may need to push come commits first)
   - "Pages > Source": Branch `gh-pages`

1. GitHub Issue labels
   - "Issues > Labels > New Label" for the following labels with color suggestions:
     - `change` (`#D93F0B`)
     - `dependency` (`#ededed`)
     - `breaking` (`#FBCA04`)

1. CodeClimate Settings
   - "Repo Settings > GitHub > Pull request status updates" (install)
   - "Repo Settings > Test coverage > Enforce {Diff,Total} Coverage" (configure to your liking)
