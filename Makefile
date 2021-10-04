.PHONY : all clean

GO ?= go
PROTOC ?= protoc
export PATH := $(shell go env GOPATH)/bin:$(PATH)

all : grpc client server
clean :
	-rm -f dspoly/*.go
	-rm -f client/client
	-rm -f server/server

grpc :
	$(PROTOC) --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		dspoly/dspoly.proto	

client : grpc client/client.go
	$(GO) build -o client/client client/client.go

server : grpc server/server.go
	$(GO) build -o server/server server/server.go
