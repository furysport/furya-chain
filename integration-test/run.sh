#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
set -x

if [[ -z "${FURYA_DAPP_REPO:-}" ]]; then
    commit=
    rm -fr furya-dapp
    git clone https://github.com/furysport/furya-dapp.git
    cd furya-dapp
    git checkout $commit
else
    cd $FURYA_DAPP_REPO
fi

yarn

npx tsx packages/scripts/integration-testing/simpleTest ..
npx tsx packages/scripts/integration-testing/upgradeTest ..