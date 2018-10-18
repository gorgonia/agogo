#!/bin/bash
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source ${SCRIPT_DIR}/../vars.sh

# envsubst doesn't exist for OSX. needs to be brew-installed
# via gettext. Should probably warn the user about that.
command -v envsubst >/dev/null 2>&1 || {
  echo >&2 "envsubst is required and not found. Aborting"
  if [[ "$OSTYPE" == "darwin"* ]]; then
    echo >&2 "------------------------------------------------"
    echo >&2 "If you're on OSX, you can install with brew via:"
    echo >&2 "  brew install gettext"
    echo >&2 "  brew link --force gettext"
  fi
  exit 1;
}

cat ${SCRIPT_DIR}/cpu-player.yaml | envsubst | kubectl apply -f -
