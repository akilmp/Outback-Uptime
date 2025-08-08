import * as k8s from '@pulumi/kubernetes';

const ns = new k8s.core.v1.Namespace('edge', {
  metadata: { name: 'edge' },
});

export const namespace = ns.metadata.name;
