#!/bin/bash

cd ape-node/cmd/ape-safer
go build -o ../../../desktop-app/bin/ape-safer
cd ../../../

cd frontend
yarn build
cd ..

rm -rf desktop-app/frontend
cp -r frontend/build desktop-app/frontend
cd desktop-app
npm run make
