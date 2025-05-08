#!/bin/bash

# It checks if the tag is annotated, otherwise it fails.
[ "$(git describe)" != "$(git describe --tags)" ] && echo "Tag must be annotated." && exit 1

CHANGELOG=$1
CURRENT_TAG=$(git describe --abbrev=0)
PREVIOUS_TAG=$(git describe --abbrev=0 "$CURRENT_TAG"^)

cat <<EOF > "$CHANGELOG"
$(git tag -l --format='%(contents)' "$CURRENT_TAG")

<details><summary><b>Changes</b></summary>

$(git log --oneline --no-decorate "$PREVIOUS_TAG".."$CURRENT_TAG")

**Full Changelog**: https://github.com/Azure/ARO-RP/compare/$PREVIOUS_TAG...$CURRENT_TAG
</details>
EOF
