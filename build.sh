#!/bin/bash

git submodule update --init --recursive
# git submodule foreach --recursive git checkout main

ROOT_DIR=$(pwd)
cd mfer-node/cmd/mfer-node
echo "Building mfer-node"
TRIPLE=$(rustc -Vv | grep host | cut -f2 -d' ')
go build -o $ROOT_DIR/mfer-safe-desktop-app/src-tauri/bin/mfer-node-$TRIPLE
cd $ROOT_DIR

echo "Building topic0"
node preprocess_topic0.js

echo "Building 4bytes"
echo "Due to too much signature files slow down computer, using Pre-built version instead"
echo "Pre-build version's commit id: 5197eb52b81b8594b6c5d3de023e649bec9523ca"
# You can build your own version by uncommenting the following lines
# git clone https://github.com/ethereum-lists/4bytes
# node preprocess_4bytes.js

cd mfer-safe-desktop-app
echo "Building desktop app"
npm i
if [ "dev" = "$1" ]; then
  npm run tauri dev
else
  npm run tauri build
fi
