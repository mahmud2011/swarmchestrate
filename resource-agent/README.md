# Resource Agent (RA)

The **Resource Agent** is the Swarmchestrate *worker-side* controller that lives on **one worker node in every cluster** expected to run user workloads. It polls the Resource Lead Agent (RLA) for work, pushes node and application heart-beats, and applies or cleans up Kubernetes manifests on its local cluster.

| Loop phase | What it does |
|------------|-------------|
| **Registration** | Registers the cluster with seed RLA, caches the RLA peer list. |
| **Node snapshot** | Sends an up-to-date worker-node inventory (CPU, memory, pressure flags…). |
| **Poll workloads** | Fetches services scheduled for *this* cluster, substitutes cross-domain placeholders, applies manifests, sets node selectors. |
| **Heart-beat** | Reports workload status (Healthy / Progressing / Failed). |
| **Cleanup** | Deletes workloads that the RLA no longer expects on this cluster. |

## ☑️ Prerequisites

- K8s / K3s clusters
- `swarmchestrate` namespace.
- kubectl

## 🚀 Deploy

```zsh
git clone https://github.com/mahmud2011/swarmchestrate.git
cd swarmchestrate/resource-agent
# Open deploy/cm.yaml
# Set `app.type` (cloud/fog/edge)
# Set `registersvc.host` with the ingress IP of the any leader or follower
kubectl apply -k deploy
```
