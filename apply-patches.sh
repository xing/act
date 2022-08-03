#!/bin/bash

set -euo pipefail

cd act || exit 1

comment=

while IFS= read -r patch
do
  case "$patch" in
    \#*)
      comment="${patch#\# }"
      ;;
    http*)
      echo
      echo "--------------------------------------------------------------------------------"
      echo "  $comment"
      echo
      echo "  Patch from: $patch"
      echo
      curl -sL "${patch/%.patch}.patch" | git am -3 || exit 1
      ;;
  esac
done < ../patches
