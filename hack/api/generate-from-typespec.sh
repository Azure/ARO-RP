#!/bin/bash -e

# VERSION determines which API version to generate.
# The default value for VERSION should be the latest API version.
# Note: typespec-go can only generate for the latest API version defined in
# the TypeSpec source. Older versions' generated code is stable and stays
# committed as-is. To regenerate older versions, use a git checkout of the
# code from when that version was the latest.
VERSION="v20250725"

cd api

target="${1:-}"
if [[ "$target" != "swagger" && "$target" != "go-api-models" && "$target" != "go-testsdk" && "$target" != "python-testsdk" && "$target" != "examples" ]]; then
    echo "Usage: $0 <swagger|go-api-models|go-testsdk|python-testsdk|examples>" >&2
    exit 1
fi

# Before generating anything, clear out existing examples to keep TypeSpec from
# complaining about conflicts
SPEC_BASE_DIR="redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters"
find "$SPEC_BASE_DIR" -mindepth 2 -maxdepth 2 -type d ! -name examples | sort | while IFS= read -r api_version_dir; do
    (
        api_version=$(basename "$api_version_dir")
        api_version_example_dir="$SPEC_BASE_DIR/examples/$api_version"
        rm -rf "$api_version_example_dir"
    )
done

npm ci
npm run format

if [[ "$target" == "swagger" || "$target" == "examples" ]]; then
    npm run swagger

    # Generate examples from Swagger. Note that when $target is "swagger", we regenerate the
    # examples and then `git restore` them. This is because oav updates the Swagger to point
    # to the examples files, and we want that to be included in a Swagger update.
    find "$SPEC_BASE_DIR" -mindepth 2 -maxdepth 2 -type d ! -name examples | sort | while IFS= read -r api_version_dir; do
        (
            api_version=$(basename "$api_version_dir")
            api_version_example_dir="$SPEC_BASE_DIR/examples/$api_version"
            mkdir -p "$api_version_example_dir"
            npm run examples -- "${api_version_dir}/redhatopenshift.json"
            mv "$api_version_dir/examples/"* "$api_version_example_dir/"
            rm -rf "$api_version_dir/examples"
        )
    done
elif [[ "$target" == "go-api-models" ]]; then
    TSP_OUTPUT_DIR="pkg/api/${VERSION}/generated"
    npm run go -- \
        --option "@azure-tools/typespec-go.emitter-output-dir={cwd}/../${TSP_OUTPUT_DIR}" \
        --option "@azure-tools/typespec-go.module=github.com/Azure/ARO-RP/${TSP_OUTPUT_DIR}"

    # Delete everything except for the models
    rm -f ../${TSP_OUTPUT_DIR}/go.mod
    rm -f ../${TSP_OUTPUT_DIR}/go.sum
    rm -f ../${TSP_OUTPUT_DIR}/LICENSE.txt
    rm -f ../${TSP_OUTPUT_DIR}/client_factory.go
    rm -f ../${TSP_OUTPUT_DIR}/*_client.go
    rm -f ../${TSP_OUTPUT_DIR}/options.go
    rm -f ../${TSP_OUTPUT_DIR}/responses.go
    rm -f ../${TSP_OUTPUT_DIR}/version.go
    rm -rf ../${TSP_OUTPUT_DIR}/testdata
elif [[ "$target" == "go-testsdk" ]]; then
    TSP_OUTPUT_DIR="pkg/client/sdk/resourcemanager/redhatopenshift/armredhatopenshift"
    npm run go -- \
        --option "@azure-tools/typespec-go.emitter-output-dir={cwd}/../${TSP_OUTPUT_DIR}" \
        --option "@azure-tools/typespec-go.module=github.com/Azure/ARO-RP/${TSP_OUTPUT_DIR}"

    # The TypeSpec Go emitter generates a few files we don't need, and there's no option to
    # disable generation of these files.
    rm -f ../${TSP_OUTPUT_DIR}/go.mod
    rm -f ../${TSP_OUTPUT_DIR}/go.sum
    rm -f ../${TSP_OUTPUT_DIR}/LICENSE.txt
elif [[ "$target" == "python-testsdk" ]]; then
    npm run python
fi

if [[ "$target" != "examples" ]]; then
    git restore "$SPEC_BASE_DIR/examples"
fi
