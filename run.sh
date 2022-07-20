#!/bin/bash

ROOT_DIR=$(pwd)
cd ape-node/cmd/ape-node
echo "Building ape-node"
TRIPLE=$(rustc -Vv | grep host | cut -f2 -d' ')
go build -o $ROOT_DIR/desktop-tauri/src-tauri/bin/ape-node-$TRIPLE
cd $ROOT_DIR

git submodule update --init --recursive
node preprocess_topic0.js

cd desktop-tauri
echo "Building desktop app"
npm i
npm run tauri dev
