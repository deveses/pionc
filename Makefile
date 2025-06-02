ifndef OS
	OS := $(shell uname -s)
endif

CGO_INCLUDE_DIR := include
CGO_LIB_DIR := lib
CGO_BUILD_DIR := $(shell mkdir -p build && realpath ./build)

#ARCH := $(shell uname -m)
CGO_OUTPUT_HEADER := libwebrtc.h

ifeq ($(OS),Windows_NT)
CGO_OUTPUT_BINARY := libwebrtc.dll
RM_CMD := del
else ifeq ($(OS),Darwin)
CGO_OUTPUT_BINARY := libwebrtc_universal.a
RM_CMD := rm -f
else ifeq ($(OS),Linux)
CGO_OUTPUT_BINARY := libwebrtc.a
RM_CMD := rm -f
else
@echo "Unsupported operating system"
endif

all: info build

info:
	@echo "Operating System: $(OS)"
	@echo "CGO Output Header: $(CGO_OUTPUT_HEADER)"
	@echo "CGO Output Binary: $(CGO_OUTPUT_BINARY)"
	@echo "CGO Include Directory: $(CGO_INCLUDE_DIR)"
	@echo "CGO Library Directory: $(CGO_LIB_DIR)"
	@echo "CGO Build Directory: $(CGO_BUILD_DIR)"
	go env

$(CGO_BUILD_DIR)/webrtc_wrapper_arm64.a:
	@echo "Building CGO library for macOS arm64..."
	cd ./src/ && CGO_ENABLED=1 GOARCH=arm64 go build -buildmode=c-archive -o ../build/webrtc_wrapper_arm64.a .
$(CGO_BUILD_DIR)/webrtc_wrapper_amd64.a:
	@echo "Building CGO library for macOS/linux amd64..."
	cd ./src/ && CGO_ENABLED=1 GOARCH=amd64 go build -buildmode=c-archive -o ../build/webrtc_wrapper_amd64.a .

$(CGO_BUILD_DIR)/$(CGO_OUTPUT_BINARY): $(CGO_BUILD_DIR)/webrtc_wrapper_arm64.a $(CGO_BUILD_DIR)/webrtc_wrapper_amd64.a
	@echo "Building CGO library for macOS universal binary..."
	cd $(CGO_BUILD_DIR) && lipo -create -output $(CGO_OUTPUT_BINARY) webrtc_wrapper_amd64.a webrtc_wrapper_arm64.a 

#vcvars32.bat
# dumpbin /EXPORTS yourfile.dll > yourfile.exports
# Paste the names of the needed functions from yourfile.exports into a new yourfile.def file. Add a line with the word EXPORTS at the top of this file.
# lib /def:yourfile.def /out:yourfile.lib
# lib /def:yourfile.def /machine:x64 /out:yourfile64.lib
# gendef.exe awesome.dll
# x86_64-w64-mingw32-dlltool.exe -d awesome.def -l awesome.lib
ifeq ($(OS),Windows_NT)
create_dirs:
	mkdir -p ./lib
	mkdir -p ./include
	mkdir -p ./build
else
create_dirs:
	mkdir -p ./lib
	mkdir -p ./include
endif

ifeq ($(OS),Windows_NT)
$(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY):
	@echo "Building CGO library for Windows..."
	go env -w CGO_ENABLED=1
# 	go env -w CC=gcc
# 	go env -w CXX=g++
	cd ./src/ && go build -v -buildmode=c-shared -o $(CGO_OUTPUT_BINARY) .

	move src\$(CGO_OUTPUT_BINARY) $(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY)
	cd lib && gendef.exe $(CGO_OUTPUT_BINARY)
	cd lib && dlltool.exe -d libwebrtc.def -l libwebrtc.lib

else ifeq ($(OS),Darwin)
$(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY): $(CGO_BUILD_DIR)/$(CGO_OUTPUT_BINARY)
	mv $(CGO_BUILD_DIR)/$(CGO_OUTPUT_BINARY) $(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY)
endif

ifeq ($(OS),Windows_NT)
$(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER):
	move src\$(CGO_OUTPUT_HEADER) ./include/$(CGO_OUTPUT_HEADER)
	powershell -ExecutionPolicy Bypass -File scripts\refine_header.ps1 -InputFile include\$(CGO_OUTPUT_HEADER) -OutputFile include\temp_header.h
	move /y include\temp_header.h include\$(CGO_OUTPUT_HEADER)
	@echo Processed $(CGO_OUTPUT_HEADER): Removed problematic block.
else ifeq ($(OS),Darwin)
$(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER):
	mv $(CGO_BUILD_DIR)/webrtc_wrapper_arm64.h ./include/libwebrtc.h
	rm $(CGO_BUILD_DIR)/webrtc_wrapper_amd64.h
else ifeq ($(OS),Linux)
$(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER):
	mv $(CGO_BUILD_DIR)/webrtc_wrapper_amd64.h ./include/libwebrtc.h
endif

build_linux: create_dirs $(CGO_BUILD_DIR)/webrtc_wrapper_amd64.a $(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER)
	mv $(CGO_BUILD_DIR)/webrtc_wrapper_amd64.a $(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY)

build_darwin: create_dirs $(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY) $(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER)

build_windows: create_dirs $(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY) $(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER)
# use gendef.exe (from mingw) to generate .def file	
# use x86_64-w64-mingw32-dlltool.exe to generate .lib file	
#	vcvars32.bat
#	cd lib && lib /def:libwebrtc.def /machine:x64 /out:libwebrtc.lib

clean:
	$(RM_CMD) $(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER)
	$(RM_CMD) $(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY)
	$(RM_CMD) $(CGO_BUILD_DIR)/*

ifeq ($(OS),Darwin)
build: build_darwin
else ifeq ($(OS),Linux)
build: build_linux
else ifeq ($(OS),Windows_NT)
build: build_windows
else
build:
	@echo "Unsupported operating system"
	exit 1
endif