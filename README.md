# Pion WebRTC Wrapper

[![Build](https://github.com/deveses/pionc/actions/workflows/build.yml/badge.svg)](https://github.com/deveses/pionc/actions/workflows/build.yml)
[![Gitter](https://img.shields.io/gitter/room/deveses/pionc)](https://matrix.to/#/#deveses-pionc:gitter.im)
[![Discord](https://img.shields.io/discord/1379253303136747661?logo=discord)](https://discord.gg/tBQC6mHW)

## Overview

This project provides a C-compatible wrapper around the Pion WebRTC library, allowing developers to use WebRTC functionalities in languages that can interface with C libraries. The wrapper exposes a set of functions for creating and managing WebRTC connections, handling media streams, and more.
Need for such a wrapper arises from the desire to use Pion's capabilities in environments where direct Go integration is not feasible, such as in C or C++ applications.
It's been used and tested in a C++ project, demonstrating its effectiveness in bridging Go's WebRTC capabilities with C/C++ applications.

## Build

1. Go project needs mod file that is created with `go mod init pionc`
1. Download dependencies: `go mod tidy`
1. Build pion wrapper: `go build -o libwebrtc.so -buildmode=c-shared webrtc_wrapper.go` 

## Usage

Initialize the Pion WebRTC library and set up callbacks for handling various WebRTC events. Below is an example of how to set up the Pion WebRTC library in a C/C++ application:

```c
PionCallbacks pionCallbacks = { 0 };
pionCallbacks.log_callback = log_callback;
pionCallbacks.remote_track_callback = WebRTCLibPeerConnection::onRemoteTrack;
pionCallbacks.ice_candidate_callback = onIceCandidate;
pionCallbacks.local_description_callback = WebRTCLibPeerConnection::onLocalDescription;
pionCallbacks.track_data_callback = WebRTCLibPeerConnection::onTrackDataCallback;
pionSetCallbacks(pionCallbacks);

PionPeerConnectionConfiguration pion_config;
pion_config.ice_servers = ice_servers.data();
pion_config.num_servers = (int)r_config.iceServers.size();
pionWebrtc = pionCreatePeerConnection(&pion_config);
```
