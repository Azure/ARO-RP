#!/bin/bash -e

#~USAGE:
#~  %cmd% <release> [options]
#~
#~ARGUMENTS:
#~  release - semver (i.e. format vX.Y.Z) formatted release to tag/publish
#~
#~OPTIONS:
#~  -b      - Switch to this branch before starting the release process (default: master)
#~  -D      - Skip the "draft" release step
#~  -p      - Create a "pre" release
#~  -n      - Perform a dry-run
#~  -s      - Sign the tag (if creating) with the default identity

REAL="$(readlink -f "$0")"
REPO="Azure/ARO-RP"
CHANGELOG="$REAL/CHANGELOG.md"
TPL_MARKER="-- everything below this line will not added to the CHANGELOG"
GITHUB_TOKEN="${GITHUB_TOKEN:-PleaseDefineMe}"
CHANGELOG_TPL="## RELEASE

This is an example of a CHANGELOG entry, edit at will.

* [BUGFIX] Fixed #ID with < some URL >
* [FEATURE] Added X, Y, Z according to < some URL >

$TPL_MARKER

Here is a list of all pull requests merged since the last release:"

# prints the usage string at the top of the script
function usage(){
  local this;
  this="$(basename "$REAL")"

  grep '^#~' "$0" |\
    sed -e "s/^#~//" |\
    sed -e "s/%cmd%/$this/"
}

# gets the last tag from local repo
function get_latest_release(){
  local this_release="$1"

  git tag --list v* |\
    grep -v "${this_release:-live}" |\
    sed -e 's/^v//' |\
    sort -n |\
    tail -1
}

# Creates a git tag
function create_tag(){
  local tag="$1"
  shift
  local sha="$1"
  shift
  local -n my_opts=$1
  local git_cmd git_opts

  git_opts="--message 'Releasing $tag'"
  if [[ "${my_opts["sign"]}" == "yes" ]]; then
    git_opts="$git_opts --sign"
  fi

  git_cmd="git tag $git_opts $tag $sha"
  if [[ "${my_opts["dry_run"]}" == "yes" ]]; then
    echo "DRY RUN: $git_cmd"
  else
    eval "$git_cmd"
  fi
}

# parse all merge commits to spit out the github PR URL and titles
function get_github_prs(){
  local range="$1"

  git log --merges --format='* %s %b' "$range" |\
    sed -e "s|Merge pull request #|https://github.com/$REPO/pull/|"
}

# creates a file with the draft to be added to the CHANGELOG
function edit_changelog_fragment(){
  local path="$1"
  local confirm="n"

  # an empty ENTER in the prompt will unset this var, but Y is the default
  until [[ "${confirm:-y}" == "y" ]]; do
    ${EDITOR:-vi} "$path"
    read -rp "Are you happy with the contents of the changelog entry (Y/n)? " choice
    confirm="$(echo "${choice:0:1}" | tr '[:upper:]' '[:lower:]')"
  done
}

# slurp the changelog fragment up to the marker
function parse_template(){
  local temp_md="$1"

  while read -r line; do
    if [[ "$line" == "$TPL_MARKER" ]]; then
      break
    fi

    echo "$line"
  done <"$temp_md"
}

# generate a payload to create a new draft in github and POST it
function create_github_release(){
  local release="$1"
  local sha="$2"
  local temp_md="$3"
  shift 3
  local -n github_opts=$1

  local payload="/tmp/payload-$release.json" pre_release="false" draft="false" curl_cmd

  if [[ "${github_opts["pre_release"]}" == "yes" ]]; then
    pre_release="true"
  fi

  if [[ "${github_opts["draft_release"]}" == "yes" ]]; then
    draft="true"
  fi

  jq -n \
    --arg body "$(parse_template "$temp_md")" \
    --arg release "$release" \
    --arg sha "$sha" \
    --argjson draft "$draft" \
    --argjson pre_release "$pre_release" \
    '{tag_name: $release, target_commitish: $sha, name: $release, body: $body, draft: $draft, prerelease: $pre_release}' \
    > "$payload"
 
  curl_cmd="curl --header \"Authorization: token $GITHUB_TOKEN\" --data @$payload --silent --verbose -X POST https://api.github.com/repos/$REPO/releases"
  if [[ "${github_opts["dry_run"]}" == "yes" ]]; then
    echo "DRY RUN: $curl_cmd"
  else
    eval "$curl_cmd"
  fi
}

# DRY
function get_sha(){
  local obj="${1:-HEAD}"
  local sha

  git rev-parse "$obj"
}

function main(){
  local this_release="v${1//v/}" # this will remove all possible "v"s in the input and leave a single one
  shift
  local -n my_opts=$1

  local temp_md="/tmp/ARO-RP-$this_release.md"
  local previous_release previous_sha this_sha

  echo "Craft release $this_release..."
  echo "Switch to ${my_opts["branch"]} first"
  git checkout "${my_opts["branch"]}"
  git pull --quiet
  previous_release="$(get_latest_release "$this_release")"
  if [[ -z "$previous_release" ]]; then
    echo "no last tag found (first release?) grabbing the last 100 commits"
    previous_release="HEAD~100"
  fi
  previous_sha="$(get_sha "$previous_release")"

  echo "Gather all PRs since last commitish $previous_release ($previous_sha)"
  echo "${CHANGELOG_TPL/RELEASE/$this_release}" > "$temp_md"
  get_github_prs "$previous_sha..$this_sha" >> "$temp_md"

  echo "Edit the changelog entry"
  edit_changelog_fragment "$temp_md"

  echo "Add fragment to CHANGELOG.md"
  if [[ "${my_opts["dry_run"]}" != "yes" ]]; then
    cat "$temp_md" >> "$CHANGELOG"
  fi

  echo "Commit changelog changes"
  if [[ "${my_opts["dry_run"]}" != "yes" ]]; then
    git add "$CHANGELOG"
    git commit --message "Updating changelog for $this_release"
  fi

  this_sha="$(get_sha)"
  create_tag "$this_release" "$this_sha" opts

  echo "Push changes and tags"
  if [[ "${my_opts["dry_run"]}" != "yes" ]]; then
    git push
    git push --tags
  fi

  echo "Create release in Github"
  create_github_release "$this_release" "$this_sha" "$temp_md" my_opts

  echo 
}

# check all required tools are there
for tool in curl jq git vi; do
  if ! command -v "$tool"; then
    echo "you are missing required tool $tool"
    exit 1
  fi
done

# show help if argument list is empty
if [[ $# -lt 1 ]]; then
  usage
  exit 0
fi

# slurp positional arguments
release="$1"
shift

# read all options
declare -A opts
opts=( ["branch"]="master" ["draft_release"]="yes" ["force"]="no" ["dry_run"]="no" ["pre_release"]="no" ["sign_tag"]="no" )

while getopts ":hb:Dfpns" flag; do
  case "$flag" in
    h)
      usage
      exit 0
      ;;
    b)
      opts["branch"]="$OPTARG"
      ;;
    D)
      opts["draft_release"]="no"
      ;;
    f)
      opts["force"]="yes"
      ;;
    p)
      opts["pre_release"]="yes"
      ;;
    n)
      opts["dry_run"]="yes"
      ;;
    s)
      opts["sign_tag"]="yes"
      ;;
    \?)
      echo "Invalid option: -${flag}"
      exit 1
      ;;
    :)
      echo "Option -${flag} requires an argument"
      exit 1
      ;;
  esac
done

export opts # quiet shellcheck
# call main func
opts["dry_run"]="yes" # being careful
main "$release" opts
unset opts
