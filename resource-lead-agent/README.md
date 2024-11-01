# Resource Lead Agent (RLA)

The **Resource Lead Agent** is the control-plane of **Swarmchestrate**. Exactly **one** RLA is elected **leader** via Raft; every additional RLA joins the same Raft cluster as a **follower**. Together they provide a *globally consistent* scheduler and REST API for every cloud, fog, and edge cluster in the system.

| Component    | Role                                                                                                 |
|--------------|-------------------------------------------------------------------------------------------------------|
| **API Server** | REST endpoints for clusters, nodes, and applications (bruno collection included). |
| **Raft Peer**  | raft–based consensus; keeps one authoritative leader and mirrors lightweight membership state. |
| **Scheduler**  | Periodic loop that places _pending_ workloads and re-queues _stalled_ ones using a pluggable scorer ; default scorer = **Borda**. |

## ☑️ Prerequisites

- K8s / K3s clusters
- Dedicated `LoadBalancer` in each cluster for the raft communication.
- `swarmchestrate` namespace.
- kubectl

## 🚀 Deploy

```zsh
git clone https://github.com/mahmud2011/swarmchestrate.git
cd swarmchestrate/resource-lead-agent
# Open deploy/cm.yaml
# Assign `raft.id` unique Raft ID for each RLA deployment.
# Populate the `raft.peers` with the `LoadBalancer` IPs.
kubectl apply -k deploy
```
