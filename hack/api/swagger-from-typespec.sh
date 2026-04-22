#!/bin/bash -e

TYPESPEC_IMAGE=$1

# Before generating Swagger and new examples, clear out existing examples to keep TypeSpec from
# complaining about conflicts
SPEC_BASE_DIR="api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters"
find "$SPEC_BASE_DIR" -mindepth 2 -maxdepth 2 -type d ! -name examples | sort | while IFS= read -r api_version_dir; do
    (
        api_version=$(basename "$api_version_dir")
        api_version_example_dir="$SPEC_BASE_DIR/examples/$api_version"
        rm -rf $api_version_example_dir
    )
done

# Format TypeSpec
docker run \
    --platform=${PLATFORM:-linux/$(go env GOARCH)} \
    --rm \
    -v $PWD/api:/api:z \
    -w /api \
    --entrypoint npm \
    "${TYPESPEC_IMAGE}" run format

# Generate Swagger from TypeSpec
docker run \
    --platform=${PLATFORM:-linux/$(go env GOARCH)} \
    --rm \
    -v $PWD/api:/api:z \
    -w /api \
    --entrypoint npm \
    "${TYPESPEC_IMAGE}" run compile

# Generate examples from Swagger
SPEC_BASE_DIR="api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters"
find "$SPEC_BASE_DIR" -mindepth 2 -maxdepth 2 -type d ! -name examples | sort | while IFS= read -r api_version_dir; do
    (
        api_version=$(basename "$api_version_dir")
        api_version_example_dir="$SPEC_BASE_DIR/examples/$api_version"
        mkdir -p $api_version_example_dir
        docker run \
            --platform=${PLATFORM:-linux/$(go env GOARCH)} \
            --rm \
            -v $PWD/api:/api:z \
            -w "/$api_version_dir" \
            --entrypoint oav \
            "${TYPESPEC_IMAGE}" generate-examples redhatopenshift.json
        mv $api_version_dir/examples/* $api_version_example_dir/
        rm -rf $api_version_dir/examples
    )
done
