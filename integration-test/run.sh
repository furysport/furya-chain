#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
set -x

if [[ -z "${FURYA_DAPP_REPO:-}" ]]; then
    rm -fr furya-app
    git clone https://github.com/furysport/furya-app.git
    cd furya-app
    git checkout $commit
else
    cd $FURYA_DAPP_REPO
fi

yarn

npx tsx packages/scripts/integration-testing/simpleTest ..
npx tsx packages/scripts/integration-testing/upgradeTest ..