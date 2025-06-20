# Swarmchestrate

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**Swarmchestrate** is a QoS-aware Kubernetes based orchestrator for **cloud-fog-edge** continuum. It turns a fleet of heterogeneous clusters into a single, self-optimising platform that deploys and migrates micro-services according to **declarative Quality-of-Service vectors** (energy, cost, performance).

* **Knowledge Base (KB)** – single source of truth (PostgreSQL by default).  
* **Resource Lead Agents (RLA)** – one Raft-elected leader + followers: REST API + global scheduler.  
* **Resource Agents (RA)** – one per worker cluster: apply manifests, send node/app heart-beats.  

---

## ☑️ Prerequisites

- KB setup. Check [knowledge-base/README.md](./knowledge-base/README.md)
- Docker
- Kind (https://kind.sigs.k8s.io) – runs local Kubernetes clusters using Docker containers.
- Kubernetes Cloud Provider for KIND (https://github.com/kubernetes-sigs/cloud-provider-kind) – provides the external IP address functionality for Kind clusters, enabling services to be exposed in a way that mimics real cloud providers.

## ⚡ Quick start

```
./bootstrap.sh all
```

- For RLA deployment check [resource-lead-agent/README.md](./resource-lead-agent/README.md)
- For RA deployment check [resource-agent/README.md](./resource-agent/README.md)
