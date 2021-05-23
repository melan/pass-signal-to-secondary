FILES := primary secondary
THIS := $(abspath $(lastword $(MAKEFILE_LIST)))

all: build

bin:
	test -d bin || mkdir bin

build: bin
build: $(FILES)

$(FILES):
	go build -o bin/$@ cmd/$@/$@.go

.PHONY: $(FILES) 