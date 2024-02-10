#!/usr/bin/env bash
cd src
go build -o reFUNDScan .
tar -I "zstd -19" -cvpf reFUNDScan-x86_64-linux.tar.zstd reFUNDScan
tar -I "gzip -9" -cvpf reFUNDScan-x86_64-linux.tar.gz reFUNDScan
rm -rf reFUNDScan
GOARCH="arm64" go build -o reFUNDScan .
tar -I "zstd -19" -cvpf reFUNDScan-aarch64-linux.tar.zstd reFUNDScan
tar -I "gzip -9" -cvpf reFUNDScan-aarch64-linux.tar.gz reFUNDScan
rm -rf reFUNDScan



