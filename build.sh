#!/bin/bash

git submodule update --init --recursive

ROOT_DIR=$(pwd)
cd mfer-node/cmd/mfer-node
echo "Building mfer-node"
TRIPLE=$(rustc -Vv | grep host | cut -f2 -d' ')
go build -o $ROOT_DIR/mfer-safe-desktop-app/src-tauri/bin/mfer-node-$TRIPLE
cd $ROOT_DIR

echo "Building topic0"
node preprocess_topic0.js

cd mfer-safe-desktop-app
echo "Building desktop app"
npm i
if [ "dev" = "$1" ]; then
  npm run tauri dev
else
  npm run tauri build
fi
