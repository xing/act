#!/bin/bash

set -euo pipefail

version=$(cd act && git describe --tags --dirty --always |cut -d "-" -f 1)
hash=$(./version-hash.sh)
versionHash=${version}-${hash}
echo "::set-output name=versionHash::${versionHash}"

gitTag=$(git for-each-ref --sort=taggerdate --format '%(tag)' refs/tags |tail -1)
# shellcheck disable=SC2001
counter=$(echo "${gitTag}" |sed -e 's/.*-xing\.\([[:digit:]]\+\)-.*/\1/')
if [ "${gitTag}"  = "${counter}" ] ; then
  counter="0"
else
  counter=$(("${counter}" + 1))
fi
tag="${version}-xing.${counter}-${hash}"
echo "::set-output name=tag::${tag}"

latestHash="$(curl -fsSLI -o /dev/null -w "%{url_effective}" https://github.com/xing/act/releases/latest)"
latestHash="${latestHash#https://github.com/xing/act/releases/tag/}"
# shellcheck disable=SC2001
latestHash=$(echo "${latestHash}" |sed -e 's/-xing\.[[:digit:]]\+//')
echo "::set-output name=latestHash::${latestHash}"

echo "versionHash: ${versionHash} tag: ${tag} latestHash: ${latestHash}"
