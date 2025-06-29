#!/bin/bash

# Simple Database script - build, push, and run PostgreSQL with L0 schema

# Load environment variables from parent script if not set
export DOCKER_REGISTRY="${DOCKER_REGISTRY:-technodamo}"
export DOCKER_REPO="${DOCKER_REPO:-wildberries}"
export DB_IMAGE="${DB_IMAGE:-${DOCKER_REGISTRY}/${DOCKER_REPO}:l0-pg}"
export DB_CONTAINER="${DB_CONTAINER:-L0-pg}"
export DB_VOLUME="${DB_VOLUME:-l0_pg_data}"
export DB_PORT="${DB_PORT:-5432}"
export DB_NAME="${DB_NAME:-L0}"
export DB_USER="${DB_USER:-task_user}"
export DB_PASSWORD="${DB_PASSWORD:-pass123!!!}"

echo "=== L0 Database Deployment ==="
echo "Image: $DB_IMAGE"
echo "Container: $DB_CONTAINER"
echo "Port: $DB_PORT"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo ""

echo "Building PostgreSQL image with L0 schema..."
docker build -t $DB_IMAGE .

echo "Pushing PostgreSQL image to Docker Hub..."
docker push $DB_IMAGE

echo "Stopping existing database containers..."
docker stop $DB_CONTAINER 2>/dev/null || true
docker rm $DB_CONTAINER 2>/dev/null || true

# Remove any container using the database port
CONTAINERS=$(docker ps -q --filter "publish=$DB_PORT")
if [ -n "$CONTAINERS" ]; then
  echo "Stopping containers using port $DB_PORT: $CONTAINERS"
  docker stop $CONTAINERS
  docker rm $CONTAINERS
fi

echo "Removing old image..."
docker rmi $DB_IMAGE 2>/dev/null || true

echo "Pulling latest PostgreSQL image from Docker Hub..."
docker pull $DB_IMAGE

echo "Creating database volume..."
docker volume create $DB_VOLUME 2>/dev/null || true

echo "Starting PostgreSQL database..."
docker run -d \
  --name $DB_CONTAINER \
  -e POSTGRES_PASSWORD=$DB_PASSWORD \
  -e POSTGRES_DB=$DB_NAME \
  -e POSTGRES_USER=$DB_USER \
  -p $DB_PORT:5432 \
  -v $DB_VOLUME:/var/lib/postgresql/data \
  $DB_IMAGE

echo "Database started! Container: $DB_CONTAINER"
echo "Port: $DB_PORT"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo "Password: $DB_PASSWORD"
echo ""
echo "To view logs: docker logs $DB_CONTAINER"
echo "To connect: psql -h localhost -p $DB_PORT -U $DB_USER -d $DB_NAME"
echo "To test connection: docker exec -it $DB_CONTAINER psql -U $DB_USER -d $DB_NAME -c '\\dt'" 