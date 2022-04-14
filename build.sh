#!/bin/bash

cd ape-node/cmd/ape-safer
echo "Building ape-node"
go build -o ../../../desktop-app/bin/ape-safer
cd ../../../

git submodule update --init --recursive
node preprocess_topic0.js

cd frontend
echo "Building frontend"
npm i
npm run build
cd ..

rm -rf desktop-app/frontend
cp -r frontend/build desktop-app/frontend
cd desktop-app
echo "Building desktop app"
npm i
npm run make
