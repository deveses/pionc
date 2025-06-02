[![Build](https://github.com/deveses/pionc/actions/workflows/build.yml/badge.svg)](https://github.com/deveses/pionc/actions/workflows/build.yml)

# Pion WebRTC Wrapper

## Overview

This project provides a C-compatible wrapper around the Pion WebRTC library, allowing developers to use WebRTC functionalities in languages that can interface with C libraries. The wrapper exposes a set of functions for creating and managing WebRTC connections, handling media streams, and more.
Need for such a wrapper arises from the desire to use Pion's capabilities in environments where direct Go integration is not feasible, such as in C or C++ applications.
It's been used and tested in a C++ project, demonstrating its effectiveness in bridging Go's WebRTC capabilities with C/C++ applications.

## Build

1. Go project needs mod file that is created with `go mod init pionc`
1. Download dependencies: `go mod tidy`
1. Build pion wrapper: `go build -o libwebrtc.so -buildmode=c-shared webrtc_wrapper.go` 