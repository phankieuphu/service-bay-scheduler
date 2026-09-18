# README

## Overview

Base code to create new another repository

---

## Documentation

* [Business Requirement](docs/business-requirement.md)
* [Architecture Design](docs/architecture-design.md)
* [Technical Requirement](docs/technical-requirement.md)
* [Responsibility Planning](docs/responsibility-planning.md)

---

## Repository Purpose

* Clean and normalize data from multiple sources
* Prepare data for banking reports
* Support extensible data ingestion (DB, Queue, etc.)
* Follow layered / hexagonal architecture

---

## Setup Guide

### Local Environment

1. Create environment variables:

```bash
cp .env.example .env
```

2. Update your local configuration in `.env`

3. Run the initialization script:

```bash
sh init.sh
``` 

---

### Docker Setup

```bash
docker compose up -d
```

---

### Kubernetes Setup

All manifests live under [k8s/](k8s/) and are wired together with a `kustomization.yaml`. Everything runs in the `service-bay` namespace.

1. Build the local images (the manifests reference `:local` tags, so a local cluster like `minikube` or `kind` needs to load them itself):

```bash
docker build -t customer-service:local ./customer-service
docker build -t vehicle-service:local ./vehicle-service
docker build -t service-bay-postgres:local ./postgres
```

If you're using `minikube`, point Docker at the cluster's daemon before building (or `minikube image load` afterwards):

```bash
eval $(minikube docker-env)
```

For `kind`, load the images into the cluster instead:

```bash
kind load docker-image customer-service:local vehicle-service:local service-bay-postgres:local
```

2. Apply all manifests via Kustomize:

```bash
kubectl apply -k k8s/
```

3. Check that everything is up:

```bash
kubectl get pods -n service-bay
```

4. Access the services (exposed as `NodePort`):

| Service          | Port  |
|------------------|-------|
| customer-service | 30080 |
| vehicle-service  | 30081 |
| prometheus       | 30090 |
| grafana          | 30030 |

With `minikube`, get a reachable URL for each with:

```bash
minikube service customer-service -n service-bay --url
```

5. Tear down:

```bash
kubectl delete -k k8s/
```

---

## Initializing a New Data Flow

### 1. Define Data Sources

#### From Database

* Implement repository adapters

#### From Queue

* Location: `internal/adapters/consumer`
* Steps:

   * Add a new consumer: `{name}Consumer.go`
   * Define input DTOs in the `/dto` folder

---

### 2. Define a New Service

1. Define service interface:

   * File: `internal/domain/ports/services.go`

2. Implement service logic:

   * Folder: `internal/domain/services`

3. Inputs & outputs:

   * Use DTOs from `internal/adapters/http` if the service is HTTP-based

---

### 3. Define Outbound Adapters (Repositories)

For database or external storage operations:

1. Define repository interface:

   * `internal/domain/ports/repositories.go`

2. Create adapter struct:

   * `internal/adapters/repositories`

3. Implement repository logic

---

## Service Architecture Layers

```
Config
  |
DB Provider
  |
Repository (Storage)
  |
Service (Use Case)
```



---



## Database Configuration

* Define database models in:

```
internal/adapters/database/models
```
---
* **Note**: if your table want to define is SQL please update file **init.sql** your SQL script

## Testing

* Write unit tests for services and repositories
* Mock external dependencies
* Run tests using standard Go **tooling**
* Run `golangci-lint run` for ensure correct syntax
---

## Deployment

* Docker-based deployment
* Environment-driven configuration
* CI/CD friendly

---

## Contribution Guidelines

* Write tests for all new features
* Follow existing code structure
* Code reviews are mandatory

---

## Contact

* Repository owner / admin
* Project team members
