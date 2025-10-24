#!/bin/bash
set -e

# Validate required environment variables
if [ -z "$REGISTRY" ]; then
    echo "ERROR: REGISTRY environment variable is not set"
    exit 1
fi

if [ -z "$BUILDER_REGISTRY" ]; then
    echo "ERROR: BUILDER_REGISTRY environment variable is not set"
    exit 1
fi

if [ -z "$VERSION" ]; then
    echo "ERROR: VERSION environment variable is not set"
    exit 1
fi

if [ -z "$LOCAL_ARO_RP_IMAGE" ]; then
    echo "ERROR: LOCAL_ARO_RP_IMAGE environment variable is not set"
    exit 1
fi

echo "Using REGISTRY=${REGISTRY}"
echo "Using BUILDER_REGISTRY=${BUILDER_REGISTRY}"
echo "Using VERSION=${VERSION}"
echo "Using LOCAL_ARO_RP_IMAGE=${LOCAL_ARO_RP_IMAGE}"

cleanup() {
    local exit_code=$?
    echo "Running cleanup..."

    if docker ps -a --format '{{.Names}}' | grep -q '^aro-test-runner$'; then
        echo "Copying test results..."
        docker cp aro-test-runner:/app/report.xml ./report.xml 2>/dev/null && echo "Copied report.xml" || echo "Could not copy report.xml"
        docker cp aro-test-runner:/app/coverage.xml ./coverage.xml 2>/dev/null && echo "Copied coverage.xml" || echo "Could not copy coverage.xml"
        docker rm aro-test-runner 2>/dev/null || true
    fi

    if docker ps -a --format '{{.Names}}' | grep -q '^local-cosmosdb$'; then
        echo "Stopping CosmosDB container..."
        docker stop local-cosmosdb 2>/dev/null || true
        docker rm local-cosmosdb 2>/dev/null || true
    fi

    if docker network ls --format '{{.Name}}' | grep -q '^ci-net$'; then
        echo "Removing Docker network ci-net..."
        docker network rm ci-net 2>/dev/null || true
    fi

    echo "Cleanup complete"
    exit $exit_code
}

trap cleanup EXIT

docker network create ci-net || true
docker run --detach -p 8081:8081 -p 10250-10255:10250-10255 --name local-cosmosdb --network ci-net mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest

echo "Waiting for CosmosDB emulator to be ready..."
timeout=180
elapsed=0
while ! curl -s -k -o /dev/null https://localhost:8081/_explorer/index.html; do
    if [ $elapsed -ge $timeout ]; then
        echo "ERROR: CosmosDB emulator failed to start after $timeout seconds"
        exit 1
    fi
    sleep 1
    elapsed=$((elapsed + 1))
done
echo "CosmosDB emulator is ready"

export DOCKER_BUILDKIT=1
echo "Building build image..."
docker build . ${DOCKER_BUILD_CI_ARGS} \
    -f Dockerfile.ci-rp-build \
    --ulimit=nofile=4096:4096 \
    --build-arg REGISTRY=${REGISTRY} \
    --build-arg BUILDER_REGISTRY=${BUILDER_REGISTRY} \
    --build-arg ARO_VERSION=${VERSION} \
    --build-arg LOCAL_COSMOS_FOR_TEST_HOST=local-cosmosdb \
    --no-cache=${NO_CACHE} \
    -t ${LOCAL_ARO_RP_IMAGE}-build:${VERSION}

echo "Build image created"


echo "Running unit tests..."
docker run --network ci-net \
    -e LOCAL_COSMOS_FOR_TEST_HOST=local-cosmosdb \
    --name aro-test-runner \
    ${LOCAL_ARO_RP_IMAGE}-build:${VERSION} \
    bash -c "gotestsum --format pkgname --junitfile /app/report.xml -- -coverprofile=/app/cover.out ./... && gocov convert /app/cover.out | gocov-xml > /app/coverage.xml"

echo "Unit tests completed"

echo "Running FIPS validation..."
docker run --rm --name aro-fips-validator ${LOCAL_ARO_RP_IMAGE}-build:${VERSION} hack/fips/validate-fips.sh ./aro

echo "FIPS validation passed"

echo "Building final slim image..."
docker build . ${DOCKER_BUILD_CI_ARGS} \
    -f Dockerfile.ci-rp \
    --build-arg REGISTRY=${REGISTRY} \
    --build-arg BUILD_IMAGE=${LOCAL_ARO_RP_IMAGE}-build:${VERSION} \
    --no-cache=${NO_CACHE} \
    -t ${LOCAL_ARO_RP_IMAGE}:${VERSION}

echo "Final image created successfully"