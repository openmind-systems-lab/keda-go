APP_VERSION ?= 1.0.0
TOPIC ?= injectMessage
SENDER_IMAGE := go-sender:$(APP_VERSION)
RECEIVER_IMAGE := go-receiver:$(APP_VERSION)

.PHONY: build build-sender build-receiver deploy deploy-kafka wait-kafka wait-kafka-api create-topic deploy-apps deploy-keda status list-topics port-forward test logs undeploy clean

build: build-sender build-receiver

build-sender:
	docker build -t $(SENDER_IMAGE) -f Dockerfile.sender .

build-receiver:
	docker build -t $(RECEIVER_IMAGE) -f Dockerfile.receiver .

deploy: deploy-kafka wait-kafka wait-kafka-api create-topic deploy-apps deploy-keda status
	@echo ""
	@echo "Deployment completed. Run: make port-forward"

# Kafka first
deploy-kafka:
	kubectl apply -f k8s/k8s-kafka.yaml

wait-kafka:
	kubectl rollout status deployment/kafka --timeout=180s

# Rollout means the container is Running, but Kafka may need a few extra seconds
# before the broker API accepts admin commands.
wait-kafka-api:
	@echo "Waiting for Kafka API..."
	@until kubectl exec deploy/kafka -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list >/dev/null 2>&1; do \
		sleep 2; \
	done

create-topic:
	kubectl exec deploy/kafka -- /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--create \
		--if-not-exists \
		--topic $(TOPIC) \
		--partitions 1 \
		--replication-factor 1

# Apps are deployed only after the topic exists.
deploy-apps:
	kubectl apply -f k8s/k8s-sender.yaml
	kubectl apply -f k8s/k8s-receiver.yaml

deploy-keda:
	kubectl apply -f k8s/k8s-scaledobject.yaml

status:
	@echo ""
	kubectl get pods
	@echo ""
	kubectl get scaledobject || true
	@echo ""
	kubectl get hpa || true

list-topics:
	kubectl exec deploy/kafka -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

port-forward:
	kubectl port-forward svc/sender-service 9999:9999

test:
	@for i in $$(seq 1 100); do \
		curl -s -X POST localhost:9999/send -H 'Content-Type: text/plain' -d "msg-$$i"; \
	done
	@echo "Sent 100 messages."

logs:
	kubectl logs -f deploy/go-receiver

undeploy:
	kubectl delete -f k8s/k8s-scaledobject.yaml --ignore-not-found
	kubectl delete -f k8s/k8s-receiver.yaml --ignore-not-found
	kubectl delete -f k8s/k8s-sender.yaml --ignore-not-found
	kubectl delete -f k8s/k8s-kafka.yaml --ignore-not-found

clean: undeploy
	docker rmi $(SENDER_IMAGE) $(RECEIVER_IMAGE) 2>/dev/null || true
