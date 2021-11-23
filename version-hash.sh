#!/bin/bash

set -euo pipefail

cd act || exit 1


temp=$(mktemp -d -t act-patches-XXXXXXXXXX)

# just test
echo 0 > "$temp/counter"

git describe --tags --dirty --always > "$temp/upstream"

grep -v '^#' < ../patches | while IFS= read -r patch
do
    curl -sL "$patch" > "$temp/${patch//\//_}"
done

hash=$(cat "$temp"/* | md5sum |cut -d " " -f 1)

rm -rf "$temp"

echo "${hash:0:7}"
