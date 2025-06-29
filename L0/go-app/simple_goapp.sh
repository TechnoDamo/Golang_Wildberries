#!/bin/bash

# Simple Go App script - build, push, and run Go application

# Load environment variables from parent script if not set
export DOCKER_REGISTRY="${DOCKER_REGISTRY:-technodamo}"
export DOCKER_REPO="${DOCKER_REPO:-wildberries}"
export GOAPP_IMAGE="${GOAPP_IMAGE:-${DOCKER_REGISTRY}/${DOCKER_REPO}:l0-goapp}"
export GOAPP_CONTAINER="${GOAPP_CONTAINER:-l0-goapp}"
export GOAPP_PORT="${GOAPP_PORT:-8080}"
export HOST_IP="${HOST_IP:-$(hostname -I | awk '{print $1}')}"
export KAFKA_PORT="${KAFKA_PORT:-9092}"
export DB_PORT="${DB_PORT:-5432}"
export DB_NAME="${DB_NAME:-L0}"
export DB_USER="${DB_USER:-task_user}"
export DB_PASSWORD="${DB_PASSWORD:-pass123!!!}"

echo "=== L0 Go Application Deployment ==="
echo "Image: $GOAPP_IMAGE"
echo "Container: $GOAPP_CONTAINER"
echo "Port: $GOAPP_PORT"
echo "Host IP: $HOST_IP"
echo "Kafka: $HOST_IP:$KAFKA_PORT"
echo "Database: $HOST_IP:$DB_PORT"
echo ""

echo "Killing existing Go App containers..."
docker stop $GOAPP_CONTAINER 2>/dev/null || true
docker rm $GOAPP_CONTAINER 2>/dev/null || true

# Remove any container using the Go app port
CONTAINERS=$(docker ps -q --filter "publish=$GOAPP_PORT")
if [ -n "$CONTAINERS" ]; then
  echo "Stopping containers using port $GOAPP_PORT: $CONTAINERS"
  docker stop $CONTAINERS
  docker rm $CONTAINERS
fi

echo "Removing old image and pulling fresh from Docker Hub..."
docker rmi $GOAPP_IMAGE 2>/dev/null || true
docker pull $GOAPP_IMAGE

echo "Starting our custom Go App..."
docker run -d \
    --name $GOAPP_CONTAINER \
    -p $GOAPP_PORT:8080 \
    -e KAFKA_BROKERS=$HOST_IP:$KAFKA_PORT \
    -e DB_HOST=$HOST_IP \
    -e DB_PORT=$DB_PORT \
    -e DB_NAME=$DB_NAME \
    -e DB_USER=$DB_USER \
    -e DB_PASSWORD=$DB_PASSWORD \
    $GOAPP_IMAGE

echo "Go App started! Container: $GOAPP_CONTAINER"
echo "Port: $GOAPP_PORT"
echo "To view logs: docker logs $GOAPP_CONTAINER"
echo "To test: curl http://localhost:$GOAPP_PORT/" 