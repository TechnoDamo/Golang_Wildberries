#!/bin/bash

export DOCKER_REGISTRY="technodamo"
export DOCKER_REPO="wildberries"
export DB_IMAGE="${DOCKER_REGISTRY}/${DOCKER_REPO}:l0-pg"
export KAFKA_IMAGE="${DOCKER_REGISTRY}/${DOCKER_REPO}:l0-kafka"
export GOAPP_IMAGE="${DOCKER_REGISTRY}/${DOCKER_REPO}:l0-goapp"

export DB_CONTAINER="l0-pg"
export KAFKA_CONTAINER="l0-kafka"
export GOAPP_CONTAINER="l0-goapp"

export DB_PORT="5432"
export KAFKA_PORT="9092"
export GOAPP_PORT="8080"

export DB_VOLUME="l0_pg_data"
export DB_NAME="L0"
export DB_USER="task_user"
export DB_PASSWORD="pass123!!!"
export KAFKA_TOPIC="orders"

export HOST_IP=$(hostname -I | awk '{print $1}')

echo "=== L0 Project Deployment ==="
echo "Host IP: $HOST_IP"
echo "Database: $DB_NAME:$DB_PORT"
echo "Kafka: $KAFKA_PORT"
echo "Go App: $GOAPP_PORT"
echo "Kafka Topic: $KAFKA_TOPIC"
echo ""

echo "Stopping existing containers..."
docker stop $DB_CONTAINER $KAFKA_CONTAINER $GOAPP_CONTAINER 2>/dev/null || true
docker rm $DB_CONTAINER $KAFKA_CONTAINER $GOAPP_CONTAINER 2>/dev/null || true

docker stop $(docker ps -q --filter "publish=$DB_PORT") 2>/dev/null || true
docker stop $(docker ps -q --filter "publish=$KAFKA_PORT") 2>/dev/null || true
docker stop $(docker ps -q --filter "publish=$GOAPP_PORT") 2>/dev/null || true
docker rm $(docker ps -aq --filter "publish=$DB_PORT") 2>/dev/null || true
docker rm $(docker ps -aq --filter "publish=$KAFKA_PORT") 2>/dev/null || true
docker rm $(docker ps -aq --filter "publish=$GOAPP_PORT") 2>/dev/null || true

echo "Building and pushing images..."
cd db && docker build -t $DB_IMAGE . && docker push $DB_IMAGE && cd ..
cd kafka && docker build -t $KAFKA_IMAGE . && docker push $KAFKA_IMAGE && cd ..
cd go-app && docker build -t $GOAPP_IMAGE . && docker push $GOAPP_IMAGE && cd ..

echo "Pulling fresh images..."
docker pull $DB_IMAGE
docker pull $KAFKA_IMAGE
docker pull $GOAPP_IMAGE

echo "Starting PostgreSQL..."
echo "Note: Database user '$DB_USER' will be created automatically by PostgreSQL with password from POSTGRES_PASSWORD"
docker volume create $DB_VOLUME 2>/dev/null || true
docker run -d --name $DB_CONTAINER \
    -e POSTGRES_PASSWORD=$DB_PASSWORD \
    -e POSTGRES_DB=$DB_NAME \
    -e POSTGRES_USER=$DB_USER \
    -p 0.0.0.0:$DB_PORT:5432 \
    -v $DB_VOLUME:/var/lib/postgresql/data \
    $DB_IMAGE

echo "Starting Kafka..."
docker run -d --name $KAFKA_CONTAINER -p 0.0.0.0:$KAFKA_PORT:9092 \
    -e KAFKA_CFG_PROCESS_ROLES=broker,controller \
    -e KAFKA_CFG_NODE_ID=1 \
    -e KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT \
    -e KAFKA_CFG_LISTENERS=PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093 \
    -e KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://$HOST_IP:$KAFKA_PORT \
    -e KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=1@localhost:9093 \
    -e KAFKA_CFG_INTER_BROKER_LISTENER_NAME=PLAINTEXT \
    -e KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER \
    -e ALLOW_PLAINTEXT_LISTENER=yes \
    $KAFKA_IMAGE

echo "Waiting for Kafka to start..."
sleep 30

echo "Creating Kafka topic: $KAFKA_TOPIC"
docker exec $KAFKA_CONTAINER kafka-topics.sh --create \
    --bootstrap-server localhost:9092 \
    --replication-factor 1 \
    --partitions 1 \
    --topic $KAFKA_TOPIC \
    --if-not-exists

echo "Starting Go Application..."
docker run -d --name $GOAPP_CONTAINER -p 0.0.0.0:$GOAPP_PORT:8080 \
    -e KAFKA_BROKERS=$HOST_IP:$KAFKA_PORT \
    -e KAFKA_TOPIC=$KAFKA_TOPIC \
    -e DB_HOST=$HOST_IP \
    -e DB_PORT=$DB_PORT \
    -e DB_NAME=$DB_NAME \
    -e DB_USER=$DB_USER \
    -e DB_PASSWORD=$DB_PASSWORD \
    $GOAPP_IMAGE

echo "Deployment complete!"
echo "Services:"
echo "  PostgreSQL: $HOST_IP:$DB_PORT"
echo "  Kafka: $HOST_IP:$KAFKA_PORT"
echo "  Go App: http://$HOST_IP:$GOAPP_PORT"
echo "  Kafka Topic: $KAFKA_TOPIC" 