#!/bin/bash
# Validate Azure DevOps pipeline YAML files

set -e

echo "Validating pipeline YAML files..."
echo "=================================="

PIPELINES=(
    ".pipelines/ci.yml"
    ".pipelines/deploy-dev-env.yml"
    ".pipelines/clean-subscription.yml"
    ".pipelines/rp-full-dev-setup.yml"
)

for pipeline in "${PIPELINES[@]}"; do
    if [ -f "$pipeline" ]; then
        echo ""
        echo "Validating: $pipeline"
        # Use yamllint if available, otherwise just check basic YAML parsing with yq/python
        if command -v yamllint &> /dev/null; then
            yamllint "$pipeline" || echo "  ⚠️  Linting warnings found"
        else
            # Basic YAML validation with Python
            python3 -c "import yaml; yaml.safe_load(open('$pipeline'))" && echo "  ✅ Valid YAML syntax" || echo "  ❌ Invalid YAML syntax"
        fi
    else
        echo "  ❌ File not found: $pipeline"
    fi
done

echo ""
echo "=================================="
echo "Validation complete!"
