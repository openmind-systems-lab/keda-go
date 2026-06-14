# KEDA Kafka Go Demo

A minimal Go + Kafka + KEDA demo.

- `go-sender`: HTTP API that publishes messages to Kafka.
- `go-receiver`: Kafka consumer scaled by KEDA from Kafka lag.
- Kafka topic: `injectMessage`.
- Consumer group: `go-receiver-group`.


## Prerequisites

- Docker Desktop with Kubernetes enabled
- `kubectl`
- KEDA installed in the cluster

Install KEDA if needed:

```bash
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm install keda kedacore/keda --namespace keda --create-namespace
```

## Quick start

```bash
make build
make deploy
```

`make deploy` runs in this order:

1. Deploy Kafka
2. Wait for the Kafka deployment
3. Wait for the Kafka broker API
4. Create the `injectMessage` topic
5. Deploy sender and receiver
6. Deploy the KEDA `ScaledObject`
7. Print status

Open a port-forward:

```bash
make port-forward
```

In another terminal, send messages:

```bash
make test
```

Watch pods with k9s:

```bash
k9s
```

Use `:po` to watch the receiver pod appear, consume messages, then disappear after KEDA cooldown.

## Useful commands

```bash
make list-topics
make logs
kubectl get scaledobject
kubectl get hpa
kubectl get deploy go-receiver
```

## Manual command sequence

Use this if you do not want to use the Makefile:

```bash
docker build -t go-sender:1.0.0 -f Dockerfile.sender .
docker build -t go-receiver:1.0.0 -f Dockerfile.receiver .

kubectl apply -f k8s/k8s-kafka.yaml
kubectl rollout status deployment/kafka --timeout=180s

until kubectl exec deploy/kafka -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list >/dev/null 2>&1; do
  sleep 2
done

kubectl exec deploy/kafka -- /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create \
  --if-not-exists \
  --topic injectMessage \
  --partitions 1 \
  --replication-factor 1

kubectl apply -f k8s/k8s-sender.yaml
kubectl apply -f k8s/k8s-receiver.yaml
kubectl apply -f k8s/k8s-scaledobject.yaml

kubectl port-forward svc/sender-service 9999:9999
```

Then send traffic:

```bash
for i in $(seq 1 100); do
  curl -s -X POST localhost:9999/send -H 'Content-Type: text/plain' -d "msg-$i"
done
```

## Cleanup

```bash
make undeploy
```

## Troubleshooting

### Pods show `ErrImageNeverPull` or `ImagePullBackOff`

This repo uses `imagePullPolicy: IfNotPresent`. On Docker Desktop Kubernetes, local images built with `docker build` should be visible to the cluster. Make sure you ran:

```bash
make build
```

before:

```bash
make deploy
```

### KEDA `READY=False`

Check operator logs:

```bash
kubectl logs -n keda deploy/keda-operator --tail=100
```

The topic is created before the ScaledObject is applied, so missing-topic errors should not happen during normal `make deploy`.
