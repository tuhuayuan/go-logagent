GOOS?=darwin
GOARCH?=amd64
OUTPUT=./build/${GOOS}/${GOARCH}

build: 
	go build -o $(OUTPUT)/logagent

clean:
	rm -rf ./build/*

.PHONY: clean, build