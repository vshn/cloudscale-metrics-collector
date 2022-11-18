# cloudscale-metrics-collector

[![Build](https://img.shields.io/github/workflow/status/vshn/cloudscale-metrics-collector/Test)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/cloudscale-metrics-collector)
[![Version](https://img.shields.io/github/v/release/vshn/cloudscale-metrics-collector)][releases]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/cloudscale-metrics-collector/total)][releases]

[build]: https://github.com/vshn/cloudscale-metrics-collector/actions?query=workflow%3ATest
[releases]: https://github.com/vshn/cloudscale-metrics-collector/releases

Batch job to sync usage data from the Cloudscale.ch metrics API to the [APPUiO Cloud reporting](https://github.com/appuio/appuio-cloud-reporting/) database.

See the [component documentation](https://hub.syn.tools/cloudscale-metrics-collector/index.html) for more information.

## Getting started

You'll need a working setup of [provider-cloudscale](https://github.com/vshn/provider-cloudscale/) and 
[appuio-cloud-reporting](https://github.com/appuio/appuio-cloud-reporting) to be able to test this collector. Make sure to follow their READMEs accordingly.

Then, set the following env variables:
```
# how many days since now metrics should be fetched from
DAYS=2

ACR_DB_URL=postgres://reporting:reporting@localhost/appuio-cloud-reporting-test
CLOUDSCALE_API_TOKEN=<API TOKEN>

# either set server url and token
KUBERNETES_SERVER_URL=<TOKEN>
KUBERNETES_SERVER_TOKEN=<TOKEN>

# or set a KUBECONFIG - this also circumvents potential x509 certificate errors when connecting to a local cluster
KUBECONFIG=/path/to/provider-cloudscale/.kind/kind-kubeconfig-v1.24.0
```

