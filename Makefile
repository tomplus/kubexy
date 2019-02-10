PROJECTNAME=$(shell basename "$(PWD)")
GOFILES=$(wildcard *.go)

all: bin/$(PROJECTNAME)

dep:
	go get -t

bin/$(PROJECTNAME): $(GOFILES) bin
	go build -o bin/$(PROJECTNAME)

clean:
	rm -fr bin/$(PROJECTNAME)
	go clean

bin:
	mkdir bin

test:
	go test -v

fmt: $(GOFILES)
	gofmt -s -w $(GOFILES)
