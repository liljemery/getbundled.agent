BINARY := getbundled-agent
CMD := ./cmd/getbundled-agent
OUT := ./bin/$(BINARY)

.PHONY: build test tidy clean

build:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(OUT) $(CMD)

build-local:
	go build -trimpath -o $(OUT) $(CMD)

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin
