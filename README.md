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

Batch job to sync usage data from the Cloudscale.ch metrics API to the [APPUiO Cloud reporting](https://github.com/appuio/appuio-cloud-reporting/) database.

## Syn Component installation

This component requires [component-appuio-cloud-reporting](https://github.com/appuio/component-appuio-cloud-reporting) and must be installed into the same
namespace. This is required for this component to be able to access the billing database and its connection secrets.

Provided that you have the [Project Syn] framework installed on a Kubernetes cluster, you can use the Commodore component included in this repository to
install the sync job.

```
applications:
  - cloudscale-metrics-collector

parameters:
  cloudscale_metrics_collector:
    namespace: 'appuio-cloud-reporting'
    secrets:
      stringData:
        cloudscale:
          token: 'TOKEN'
```

You need to get the token from the [Cloudscale Control Panel](https://control.cloudscale.ch). You need to select the correct Project (token is limited to one
project), choose "API Tokens" in the menu and generate a new one.

Since this is applied to the cluster globally, it has to go into the Commodore defaults repo (e.g. https://git.vshn.net/syn/commodore-defaults/). 
