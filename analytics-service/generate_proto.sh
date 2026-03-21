#!/usr/bin/env bash
# Run from analytics-service/ root
# Requires: pip install grpcio-tools

python -m grpc_tools.protoc \
  -I./proto \
  --python_out=./generated \
  --grpc_python_out=./generated \
  proto/analytics.proto

echo "Generated files in ./generated/"
