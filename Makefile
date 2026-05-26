server:
	cd cmd/server && go run server.go
server_test:
	cd cmd/server && go test -v
client:
	cd cmd/client && go run client.go $(NAME)