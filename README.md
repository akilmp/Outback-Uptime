# Outback Uptime – Multi‑Cloud Edge‑to‑Core DevOps Pipeline for Remote IoT Stations

## Table of Contents

1. [Project Overview](#project-overview)
2. [System Architecture](#system-architecture)
3. [Skill‑to‑Component Mapping](#skill-to-component-mapping)
4. [Tech Stack](#tech-stack)
5. [Data & Traffic Flow](#data--traffic-flow)
6. [Repository Structure](#repository-structure)
7. [Prerequisites](#prerequisites)
8. [Local Development & Demo Environment](#local-development--demo-environment)
9. [Infrastructure‑as‑Code](#infrastructure-as-code)
10. [GitOps & Configuration Management](#gitops--configuration-management)
11. [CI Pipeline](#ci-pipeline)
12. [Progressive Delivery (Argo Rollouts)](#progressive-delivery-argo-rollouts)
13. [Observability & SLOs](#observability--slos)
14. [Chaos Engineering & DR](#chaos-engineering--dr)
15. [Security & Policy‑as‑Code](#security--policy-as-code)
16. [Cost Management](#cost-management)
17. [Troubleshooting & FAQ](#troubleshooting--faq)
18. [Stretch Goals](#stretch-goals)
19. [References](#references)

---

## Project Overview

**Outback Uptime** is a DevOps/SRE showcase that keeps solar‑powered IoT sensor hubs on remote Australian cattle stations online with **zero‑downtime**, even across simultaneous cloud‑region failures. Data streams via MQTT to an edge K3s cluster, tunnels to dual‑cloud back‑ends (AWS EKS in Sydney & Azure AKS in Melbourne), and is served through a Linkerd zero‑trust mesh. Everything is defined as code (Terraform + Pulumi), delivered GitOps‑style with Flux CD and Argo Rollouts, monitored by Prometheus/Loki/Grafana, and tested under chaos (Litmus) with cost guard‑rails (Kubecost & Infracost). Idle cost ≈ AUD 24 / month.

---

## System Architecture

```text
               +-------------+ 4G/Starlink  +--------------------+
Solar IoT Hub ➜| MQTT Broker |──────────────▶| Edge K3s Cluster   |
               +-------------+               |  mqtt‑bridge       |
                                             |  skupper tunnel    |
                                             +----------+---------+
                                                        |
                                                        ▼
     (AMQP Tunnel)              STRICT mTLS (Istio)              (Pub/Sub Relay)
+--------------------+    +--------------------------------+    +--------------------+
|  AWS EKS Sydney    | ⇆  |  East‑West Gateway (Istio)     | ⇆  |  Azure AKS Melbourne|
|  ingest-api svc    |    +--------------------------------+    |  ingest-api svc     |
|  ClickHouse + OTel |                                        |  ClickHouse + OTel  |
+---------+----------+                                        +----------+-----------+
          ▲                                                          ▲
          | Velero backup ➜ S3     Global DNS (NS1)      Velero backup ➜ Azure Blob
          | (S3 Glacier tier)             ▲                         |
          +-------------------------------┴-------------------------+
```

*Fail‑over path*: NS1 Pulsar health checks set traffic weights; chaos script kills AWS ingress → Pulsar routes clients to Azure in < 30 s.

---

## Skill‑to‑Component Mapping

| DevOps/SRE Skill                | Outback Uptime Implementation                                         |
| ------------------------------- | --------------------------------------------------------------------- |
| IaC & Modular Design            | Terraform (core VPC, clusters) + Pulumi (edge K3s)                    |
| GitOps                          | Flux CD syncs Helm charts; Kustomize overlays per env                 |
| CI with Security & FinOps gates | GitHub Actions → Go tests, Trivy image scan, Infracost comment        |
| Progressive Delivery            | Argo Rollouts canary & blue/green with Prometheus metrics analysis    |
| Service Mesh Zero‑Trust         | Linkerd 2.15 automatic mTLS; tap CLI demo                             |
| Observability                   | kube‑prom‑stack (Prometheus, Loki, Grafana) + OpenTelemetry traces    |
| Chaos & Resilience              | LitmusChaos experiments (network drop, node kill); Velero backups     |
| Secrets & Policy                | Vault Agent injector; OPA Gatekeeper (image provenance, no‑root pods) |
| Cost Optimisation               | Karpenter mixed‑instance spot pool; Kubecost alerting                 |

---

## Tech Stack

| Category      | Tool / Service                                 |
| ------------- | ---------------------------------------------- |
| IaC           | Terraform 1.7, Pulumi v4 (TypeScript)          |
| GitOps        | Flux CD 2, Kustomize 5                         |
| CI/CD         | GitHub Actions, Argo Rollouts 1.6              |
| Mesh          | Linkerd 2.15, Skupper tunnel (edge↔core)       |
| Observability | Prometheus 2.51, Grafana 11, Loki 2.9, Tempo 2 |
| Chaos         | LitmusChaos 3.0                                |
| DR/Backup     | Velero 1.13 (restic)                           |
| AuthN/Z       | Vault 1.15, OPA Gatekeeper 3.12                |
| Database      | ClickHouse 23 (hot OLAP), S3 Glacier archive   |
| DNS/Fail‑over | NS1 Pulsar weighted DNS                        |
| Cost          | Kubecost 2.0, Infracost 0.10                   |

---

## Data & Traffic Flow

1. **Edge MQTT** sensor packets (JSON) every 30 s.
2. `mqtt‑bridge` publishes to Skupper AMQP tunnel.
3. `ingest-api` Go service consumes, deduplicates, writes to ClickHouse.
4. OpenTelemetry sidecar exports traces & metrics.
5. Grafana live heat‑map queries ClickHouse materialised view.

Latency (edge → ClickHouse commit) P95 < 300 ms.

---

## Repository Structure

```
outback-uptime/
├── edge/
│   ├── pulumi/                # Pulumi TS stack for K3s on Pi
│   └── k3s-manifests/         # mqtt‑bridge, skupper, cert‑manager
├── apps/
│   └── ingest-api/            # Go micro‑service
├── charts/                    # Helm charts (ingest-api, clickhouse, linkerd)
├── infra/
│   ├── terraform/
│   │   ├── aws/               # VPC, EKS, Kinesis
│   │   └── azure/             # VNet, AKS, Traffic Manager
│   └── helmfile/              # mesh, monitoring, rollouts, vault
├── argo-rollouts/
│   └── ingest-rollout.yaml
├── flux/
│   ├── root.yaml              # app-of-apps
│   └── overlays/
│       ├── dev/
│       └── prod/
├── policy/                    # Gatekeeper constraints
├── litmus/                    # chaos experiments CRDs
├── hack/
│   └── kill-ingress.sh        # fail-over drill
└── .github/workflows/
    ├── ci.yml
    └── rollout-update.yml
```

---

## Prerequisites

* AWS CLI v2 + profile `outback-aws`
* Azure CLI + subscription owner
* kubectl 1.29, helm 3, linkerd cli
* Terraform 1.7 & Pulumi >= 4
* NS1 API key, Vault dev token
* Raspberry Pi 4 (4 GB) or x86 VM for edge demo (optional)

---

## Local Development & Demo Environment

```bash
# set up Go & run unit tests
cd apps/ingest-api && go test ./...

# run ingest-api locally
go run cmd/server/main.go --dry-run

# spin kind cluster + Linkerd demo
kind create cluster --name outback-dev
linkerd install | kubectl apply -f -
linkerd check
```

Edge emulation: `docker compose -f edge/docker-compose.sim.yaml up` publishes MQTT messages.

### Edge Deployment & Failover Drill

```bash
# apply edge namespace workloads
kubectl apply -f edge/k3s-manifests/

# shift traffic from primary to secondary (dry run)
DRY_RUN=true RESOURCE_GROUP=my-rg PROFILE_NAME=my-tm \
  hack/failover.sh aws azure
```

---

## Infrastructure‑as‑Code

1. `infra/terraform/aws` — VPC, EKS, S3, IAM OIDC, Karpenter.
2. `infra/terraform/azure` — VNet, AKS Autopilot, Traffic Manager, Storage.
3. `edge/pulumi` — K3s install via k3sup, fleet Helm releases.

To provision the edge stack:

```bash
cd edge/pulumi
npm install
pulumi config set brokerUrl <mqtt-url>
pulumi config set brokerUser <username>
pulumi config set --secret brokerPassword <password>
pulumi up
```

The command outputs the edge namespace and Helm release names for the deployed charts.

CI plans run Infracost & tfsec before merge.

---

## GitOps & Configuration Management

* **Flux CD** watches `flux/root.yaml` → deploys Argo Rollouts, mesh, monitoring.
* **Kustomize overlays** add env‑specific patches (replicas, resource limits).
* **SealedSecrets** for non‑Vault edge configs.

---

## CI Pipeline (GitHub Actions)

```yaml
name: ci
on: push
jobs:
  test-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go test ./...
      - name: Trivy scan
        uses: aquasecurity/trivy-action@v0.12.0
        with:
          image-ref: ghcr.io/${{github.repository}}/ingest-api:${{github.sha}}
      - name: Infracost comment
        uses: infracost/actions@v2
```

---

## Progressive Delivery (Argo Rollouts)

* Rollout manifest defines analysis template querying Prometheus metric `ingest_latency_p95`.
* Steps: 10 % → pause 2 min → 50 % → pause 5 min → full.
* Auto‑abort if latency increase > 20 % baseline.
* Monitor rollout with `kubectl argo rollouts get rollout ingest-api -w -n <env>` or via the Argo Rollouts dashboard.
* Roll back instantly with `kubectl argo rollouts undo ingest-api -n <env>`.

---

## Observability & SLOs

| Metric           | Source           | SLO / Alert                           |
| ---------------- | ---------------- | ------------------------------------- |
| `ingest_latency` | OTel exporter    | P99 < 500 ms                          |
| `ingest_errors`  | Prometheus meter | Error rate < 1 % (burn over 10 min)   |
| `click_cpu`      | node\_exporter   | Alert if > 70 % for 15 min (scale up) |
| `cost_cluster`   | Kubecost API     | Warn > \$30 / mo projection           |

Grafana dashboards exported to `docs/grafana/outback_dash.json`.

---

## Chaos Engineering & DR

* **Litmus** DNS chaos + node kill.
* **Chaos workflow**: run experiment nightly in dev; PagerDuty on failure.
* **Velero** scheduled backups every 6 h with restic; restore tested monthly.

Fail‑over drill: `hack/kill-ingress.sh` + NS1 Pulsar weight shift.

---

## Security & Policy‑as‑Code

* Vault Agent sidecar injects DB creds; rotation 30 d.
* Gatekeeper constraints:

  * `no-root-containers`
  * `image-must-be-scanned`
  * `team-label-required`
* Image provenance: cosign + Rekor, verified by policy.

---

## Cost Management

| Component  | Optimisation          | Idle AUD/mo |
| ---------- | --------------------- | ----------- |
| EKS nodes  | Spot + Karpenter      | 12          |
| AKS pods   | Autopilot pay‑per‑pod | 8           |
| ClickHouse | t4g.small gp3 EBS     | 3           |
| DNS / NS1  | Dev tier              | 1           |
| Monitoring | Grafana Cloud free    | 0           |
| **Total**  |                       | **24**      |

Alert: Kubecost slack when daily spend > \$1.

---




## Troubleshooting & FAQ

| Issue                     | Fix                                                                      |
| ------------------------- | ------------------------------------------------------------------------ |
| Flux “image pull backoff” | Check Vault inject annotation & ECR policy                               |
| Rollout aborts at 10 %    | Ensure Prometheus analysis query returns <8 s and baseline label correct |
| Skupper tunnel drops      | Verify edge public IP changed; restart `kubectl port-forward` agent      |
| Velero restore failed     | Restic repo locked; run `velero restic unlock`                           |

---

## Stretch Goals

* **Cilium + Hubble** for eBPF observability.
* **SPIRE / SPIFFE** workload identities across cloud.
* **Backstage portal** for service templates.
* **OpenTF modules** for maintainability.

---

## References

* Karpenter docs 2025
* Linkerd security best practices 2025
* Argo Rollouts analysis templates 2024
* LitmusChaos hub – [https://hub.litmuschaos.io/](https://hub.litmuschaos.io/)
* Kubecost & Infracost integration guide 2025

---

*Last updated: 4 Aug 2025*
