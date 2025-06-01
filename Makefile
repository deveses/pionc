ifndef OS
	OS := $(shell uname -s)
endif

CGO_INCLUDE_DIR := include
CGO_LIB_DIR := lib

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


all: build

webrtc_wrapper_arm64.a:
	cd ./src/ && CGO_ENABLED=1 GOARCH=arm64 go build -buildmode=c-archive -o webrtc_wrapper_arm64.a .
webrtc_wrapper_amd64.a:
	cd ./src/ && CGO_ENABLED=1 GOARCH=amd64 go build -buildmode=c-archive -o webrtc_wrapper_amd64.a .

webrtc_wrapper_universal.a: webrtc_wrapper_arm64.a webrtc_wrapper_amd64.a
	cd ./src/ && lipo -create -output webrtc_wrapper_universal.a webrtc_wrapper_amd64.a webrtc_wrapper_arm64.a 

#vcvars32.bat
# dumpbin /EXPORTS yourfile.dll > yourfile.exports
# Paste the names of the needed functions from yourfile.exports into a new yourfile.def file. Add a line with the word EXPORTS at the top of this file.
# lib /def:yourfile.def /out:yourfile.lib
# lib /def:yourfile.def /machine:x64 /out:yourfile64.lib
# gendef.exe awesome.dll
# x86_64-w64-mingw32-dlltool.exe -d awesome.def -l awesome.lib
create_dirs:
 ifeq ($(OS),Windows_NT)
	@if not exist "lib" mkdir lib
	@if not exist "include" mkdir include
 else
	mkdir -p ./lib
	mkdir -p ./include
 endif

build_darwin: create_dirs webrtc_wrapper_universal.a
	mv ./src/webrtc_wrapper_universal.a ./lib/libwebrtc_universal.a
	mv ./src/webrtc_wrapper_arm64.h ./include/libwebrtc.h
	rm ./src/webrtc_wrapper_amd64.h

build_linux: create_dirs webrtc_wrapper_amd64.a
	mv ./src/webrtc_wrapper_amd64.a ./lib/libwebrtc.a
	mv ./src/webrtc_wrapper_amd64.h ./include/libwebrtc.h

$(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY):
	go env -w CGO_ENABLED=1
# 	go env -w CC=gcc
# 	go env -w CXX=g++
	cd ./src/ && go build -v -buildmode=c-shared -o $(CGO_OUTPUT_BINARY) .

	move src\$(CGO_OUTPUT_BINARY) $(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY)
	cd lib && gendef.exe $(CGO_OUTPUT_BINARY)
	cd lib && dlltool.exe -d libwebrtc.def -l libwebrtc.lib

$(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER):
	move src\$(CGO_OUTPUT_HEADER) ./include/$(CGO_OUTPUT_HEADER)
	powershell -ExecutionPolicy Bypass -File scripts\refine_header.ps1 -InputFile include\$(CGO_OUTPUT_HEADER) -OutputFile include\temp_header.h
	move /y include\temp_header.h include\$(CGO_OUTPUT_HEADER)
	@echo Processed $(CGO_OUTPUT_HEADER): Removed problematic block.

build_windows: create_dirs $(CGO_LIB_DIR)/$(CGO_OUTPUT_BINARY) $(CGO_INCLUDE_DIR)/$(CGO_OUTPUT_HEADER)
# use gendef.exe (from mingw) to generate .def file	
# use x86_64-w64-mingw32-dlltool.exe to generate .lib file	
#	vcvars32.bat
#	cd lib && lib /def:libwebrtc.def /machine:x64 /out:libwebrtc.lib

clean:
	$(RM_CMD) $(CGO_INCLUDE_DIR)\$(CGO_OUTPUT_HEADER)
	$(RM_CMD) $(CGO_LIB_DIR)\$(CGO_OUTPUT_BINARY)

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