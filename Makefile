.PHONY: build/hi
build/hi: main.go hello/service.pb.go
	GOOS=linux GOARCH=amd64 go build -o build/hi

hello/service.pb.go: hello/service.proto
	protoc -I=./hello --go_out=plugins=grpc:./hello service.proto
