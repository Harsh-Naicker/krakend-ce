.PHONY: all backend client clean

all: backend client

backend:
	go build -o sse_backend main.go
	./sse_backend

client:
	go build -o sse_client client/client.go

clean:
	rm -f sse_backend sse_client 