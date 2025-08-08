import * as pulumi from '@pulumi/pulumi';
import * as k8s from '@pulumi/kubernetes';
import * as command from '@pulumi/command';
import * as fs from 'fs';

const config = new pulumi.Config();

const brokerUrl = config.require('brokerUrl');
const brokerUser = config.require('brokerUser');
const brokerPassword = config.requireSecret('brokerPassword');

const installK3s = config.getBoolean('installK3s') || false;
const kubeconfigPath =
  config.get('kubeconfig') || '/etc/rancher/k3s/k3s.yaml';

let provider: k8s.Provider;

if (installK3s) {
  const k3s = new command.local.Command('install-k3s', {
    create:
      "curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC='server --write-kubeconfig-mode 644' sh -",
  });
  provider = new k8s.Provider('k3s', {
    kubeconfig: k3s.stdout.apply(() =>
      fs.readFileSync(kubeconfigPath).toString(),
    ),
  });
} else {
  provider = new k8s.Provider('k3s', {
    kubeconfig: fs.readFileSync(kubeconfigPath).toString(),
  });
}

const ns = new k8s.core.v1.Namespace(
  'edge',
  {
    metadata: { name: 'edge' },
  },
  { provider },
);

const mqttBridge = new k8s.helm.v3.Chart(
  'mqtt-bridge',
  {
    path: '../../charts/mqtt-bridge',
    namespace: ns.metadata.name,
    values: {
      broker: {
        url: brokerUrl,
        username: brokerUser,
        password: brokerPassword,
      },
    },
  },
  { provider },
);

const skupper = new k8s.helm.v3.Release(
  'skupper',
  {
    chart: 'skupper',
    repositoryOpts: { repo: 'https://skupper.io/helm' },
    namespace: ns.metadata.name,
  },
  { provider },
);

const certManager = new k8s.helm.v3.Release(
  'cert-manager',
  {
    chart: 'cert-manager',
    repositoryOpts: { repo: 'https://charts.jetstack.io' },
    namespace: 'cert-manager',
    values: {
      installCRDs: true,
    },
  },
  { provider },
);

export const namespace = ns.metadata.name;
export const mqttBridgeService = mqttBridge
  .getResource('v1/Service', 'edge/mqtt-bridge')
  .metadata.name;
export const skupperRelease = skupper.name;
export const certManagerRelease = certManager.name;
export const kubeconfig = kubeconfigPath;
