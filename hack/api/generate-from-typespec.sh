#!/bin/bash -e

cd api

target="${1:-}"
if [[ "$target" != "swagger" && "$target" != "python" && "$target" != "examples" ]]; then
    echo "Usage: $0 <swagger|python|examples>" >&2
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
elif [[ "$target" == "python" ]]; then
    npm run python
fi

if [[ "$target" != "examples" ]]; then
    git restore "$SPEC_BASE_DIR/examples"
fi
