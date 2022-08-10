local kap = import 'lib/kapitan.libjsonnet';
local inv = kap.inventory();
local params = 'inv.parameters.cloudscale-metrics-collector';
local argocd = import 'lib/argocd.libjsonnet';

local app = argocd.App('cloudscale-metrics-collector', params.namespace, secrets=false);

{
  'cloudscale-metrics-collector': app,
}
