local kap = import 'lib/kapitan.libjsonnet';
local inv = kap.inventory();
local params = inv.parameters.cloudscale_metrics_collector;
local paramsACR = inv.parameters.appuio_cloud_reporting;
local kube = import 'lib/kube.libjsonnet';
local com = import 'lib/commodore.libjsonnet';
local collectorImage = '%(registry)s/%(repository)s:%(tag)s' % params.images.collector;


local labels = {
  'app.kubernetes.io/name': 'appuio-cloud-reporting',
  'app.kubernetes.io/managed-by': 'commodore',
  'app.kubernetes.io/part-of': 'syn',
};

local secrets = [
  if params.secrets[s] != null then
    kube.Secret(s) {
      metadata+: {
        namespace: paramsACR.namespace,
      }
    } + com.makeMergeable(params.secrets[s])
  for s in std.objectFields(params.secrets)
];

{
  assert params.secrets != null : 'secrets must be set.',
  assert params.secrets.cloudscale != null : 'secrets.cloudscale must be set.',
  assert params.secrets.cloudscale.stringData != null : 'secrets.cloudscale.stringData must be set.',
  assert params.secrets.cloudscale.stringData.token != null : 'secrets.cloudscale.stringData.token must be set.',
  secrets: std.filter(function(it) it != null, secrets),

  cronjob: {
    kind: 'CronJob',
    apiVersion: 'batch/v1',
    metadata: {
      name: 'cloudscale-metrics-collector',
      namespace: paramsACR.namespace,
      labels+: labels,
    },
    spec: {
      concurrencyPolicy: 'Forbid',
      failedJobsHistoryLimit: 5,
      jobTemplate: {
        spec: {
          template: {
            spec: {
              restartPolicy: 'OnFailure',
              containers: [
                {
                  args: [
                    'cloudscale-metrics-collector',
                  ],
                  command: ['sh', '-c'],
                  env: [
                    {
                      name: 'password',
                      valueFrom: {
                        secretKeyRef: {
                          key: 'password',
                          name: 'reporting-db',
                        },
                      },
                    },
                    {
                      name: 'username',
                      valueFrom: {
                        secretKeyRef: {
                          key: 'username',
                          name: 'reporting-db',
                        },
                      },
                    },
                    {
                      name: 'ACR_DB_URL',
                      value: 'postgres://$(username):$(password)@%(host)s:%(port)s/%(name)s?%(parameters)s' % paramsACR.database,
                    },
                    {
                      name: 'CLOUDSCALE_API_TOKEN',
                      valueFrom: {
                        secretKeyRef: {
                          key: 'token',
                          name: 'cloudscale',
                        },
                      },
                    },
                  ],
                  image: collectorImage,
                  name: 'cloudscale-metrics-collector-backfill',
                  resources: {},
                },
              ],
            },
          },
        },
      },
      schedule: '10 4,10,16,22 * * *',
      successfulJobsHistoryLimit: 3,
    },
  },
}
