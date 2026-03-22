#!/usr/bin/env bash
# Run from analytics-service/ root
# Requires: pip3 install grpcio-tools

python3 -m grpc_tools.protoc \
  -I./proto \
  --python_out=. \
  --grpc_python_out=. \
  proto/analytics.proto
