#!/bin/bash

set -euxo pipefail

cd act || exit 1

grep -v '^#' < ../patches | while IFS= read -r patch
do
    echo "Applying patch $patch"
    curl -L "$patch" | git am -3 || exit 1
done
