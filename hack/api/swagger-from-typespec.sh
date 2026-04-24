#!/bin/bash -e

cd api

# Before generating Swagger and new examples, clear out existing examples to keep TypeSpec from
# complaining about conflicts
SPEC_BASE_DIR="redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters"
find "$SPEC_BASE_DIR" -mindepth 2 -maxdepth 2 -type d ! -name examples | sort | while IFS= read -r api_version_dir; do
    (
        api_version=$(basename "$api_version_dir")
        api_version_example_dir="$SPEC_BASE_DIR/examples/$api_version"
        rm -rf $api_version_example_dir
    )
done

npm install
npm run format
npm run compile

# Generate examples from Swagger
find "$SPEC_BASE_DIR" -mindepth 2 -maxdepth 2 -type d ! -name examples | sort | while IFS= read -r api_version_dir; do
    (
        api_version=$(basename "$api_version_dir")
        api_version_example_dir="$SPEC_BASE_DIR/examples/$api_version"
        mkdir -p $api_version_example_dir
        npm run examples -- ${api_version_dir}/redhatopenshift.json
        mv $api_version_dir/examples/* $api_version_example_dir/
        rm -rf $api_version_dir/examples
    )
done
