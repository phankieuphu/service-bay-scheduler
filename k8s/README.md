# Kubernetes manifests

Mirrors `docker-compose.yml` for a local cluster (minikube/kind). All resources live in the `service-bay` namespace.

## Build the images the cluster needs

The manifests reference locally-built images, not a registry:

```sh
docker build -t customer-service:local ./customer-service
docker build -t vehicle-service:local ./vehicle-service
docker build -t service-bay-postgres:local ./postgres
```

Load them into your cluster (kind example — minikube uses `minikube image load`):

```sh
kind load docker-image customer-service:local vehicle-service:local service-bay-postgres:local
```

## Apply

```sh
kubectl apply -k k8s/
```

## Access

NodePort services are exposed on the cluster node's IP:

- customer-service: 30080
- vehicle-service: 30081
- prometheus: 30090
- grafana: 30030 (admin/admin)
- flink web UI: 30082

postgres, zookeeper, kafka and redis are ClusterIP-only, matching their internal-only role in `docker-compose.yml`.
