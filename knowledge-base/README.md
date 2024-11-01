# Knowledge Base

The knowledge-base is a data store designed to store information of clusters, nodes and applications. This PostgreSQL based approach is taken into account to simplify the process. The knowledge-base can be blockchain or replicated SQL or NoSQL instance.

## ☑️ Prerequisites

- Docker
- Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate)

## Installation

```zsh
git clone https://github.com/mahmud2011/swarmchestrate.git
./bootstrap.sh kb
```

## Usage

- The PostgreSQL database will be available on port `5432`.
- pgAdmin will be available on port `8089`.

## Connecting to pgAdmin

To connect to the PostgreSQL database using pgAdmin:

- Open pgAdmin in a browser at [http://localhost:8089](http://localhost:8089)
- Use the following credentials:
  - **Email:** `foo@bar.com`
  - **Password:** `pass`
- Add a new server with the following details:
  - **Host:** `host.docker.internal`
  - **Port:** `5432`
  - **Username:** `foo`
  - **Password:** `pass`
  - **Database:** `knowledge_base`
