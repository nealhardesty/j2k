# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOTEST=$(GOCMD) test

CGO_ENABLED=1 
# $env:CGO_ENABLED=1; go run demo.go

BINARY_NAME=j2k.exe

build:
	$(GOBUILD) -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Initial init
init:
	choco install mingw -y
	$(GOMOD) init github.com/nealhardesty/j2k

# Update Go modules
mod:
	$(GOMOD) tidy
	$(GOMOD) vendor

# Run tests
test:
	$(GOTEST) -v ./...

.PHONY: build run clean mod test
