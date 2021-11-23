#!/bin/bash

set -euo pipefail

cd act || exit 1

grep -v '^#' < ../patches | while IFS= read -r patch
do
    echo "Patch from: $patch"
    curl -sL "$patch" | git am -3 || exit 1
done
