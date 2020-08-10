server:
	go run framework/cmd/server/server.go

test:
	go test -cover ./...

.PHONY: server test