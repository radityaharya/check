#!/bin/bash
set -e

cd "$(dirname "$0")/.."

if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed"
    echo "Install it from: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

mkdir -p proto/pb

cd proto
protoc --go_out=pb \
       --go_opt=paths=source_relative \
       --go-grpc_out=pb \
       --go-grpc_opt=paths=source_relative \
       monitor.proto
cd ..

echo "Proto code generated successfully in proto/pb/"

