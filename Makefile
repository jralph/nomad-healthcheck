.SILENT: setup clean
.PHONY: setup clean

build:
	make clean && make setup && make compile

compile:
	GOOS="linux" GOARCH="amd64" go build -o "bin/nomad-healthcheck-linux-amd64" nomad-healthcheck.go
	GOOS="linux" GOARCH="386" go build -o "bin/nomad-healthcheck-linux-386" nomad-healthcheck.go
	GOOS="darwin" GOARCH="amd64" go build -o "bin/nomad-healthcheck-darwin-amd64" nomad-healthcheck.go
	GOOS="darwin" GOARCH="386" go build -o "bin/nomad-healthcheck-dawrin-386" nomad-healthcheck.go

setup:
	mkdir bin

clean:
	rm -rf bin
