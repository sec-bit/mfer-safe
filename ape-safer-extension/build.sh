#!/bin/bash

ORIG_PATH=`pwd`
cd eip1193-provider 
npm run build
cd $ORIG_PATH
cp eip1193-provider/dist/umd/index.min.js public/provider.js

INLINE_RUNTIME_CHUNK=false yarn build