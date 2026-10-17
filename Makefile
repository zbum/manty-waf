APP := manty-waf
CMD := ./cmd/manty-waf
DIST := dist
LISTEN ?= :8080
UPSTREAM ?= http://127.0.0.1:8081

.PHONY: build test build-all clean run

build:
	mkdir -p $(DIST)
	go build -o $(DIST)/$(APP) $(CMD)

test:
	go test ./...

run:
	go run $(CMD) -listen $(LISTEN) -upstream $(UPSTREAM)

build-all:
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 go build -o $(DIST)/$(APP)-darwin-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -o $(DIST)/$(APP)-darwin-arm64 $(CMD)
	GOOS=linux GOARCH=amd64 go build -o $(DIST)/$(APP)-linux-amd64 $(CMD)
	GOOS=linux GOARCH=arm64 go build -o $(DIST)/$(APP)-linux-arm64 $(CMD)
	GOOS=windows GOARCH=amd64 go build -o $(DIST)/$(APP)-windows-amd64.exe $(CMD)
	GOOS=windows GOARCH=arm64 go build -o $(DIST)/$(APP)-windows-arm64.exe $(CMD)

clean:
	rm -rf $(DIST)
