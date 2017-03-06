GOOS?=darwin 
GOARCH?=amd64

build:
	go build 

clean:
	rm ./logagent

.PHONY: clean