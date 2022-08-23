local kap = import 'lib/kapitan.libjsonnet';
local inv = kap.inventory();
local params = inv.parameters.cloudscale_metrics_collector;
local paramsACR = inv.parameters.appuio_cloud_reporting;
local argocd = import 'lib/argocd.libjsonnet';

local app = argocd.App('cloudscale-metrics-collector', paramsACR.namespace);

{
  'cloudscale-metrics-collector': app,
}
