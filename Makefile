GOMOD=$(shell test -f "go.work" && echo "readonly" || echo "vendor")
LDFLAGS=-s -w

cli:
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/server cmd/server/main.go

lambda:
	@make lambda-server

lambda-server:
	if test -f bootstrap; then rm -f bootstrap; fi
	if test -f server.zip; then rm -f server.zip; fi
	GOARCH=arm64 GOOS=linux go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -tags lambda.norpc -o bootstrap cmd/server/main.go
	zip server.zip bootstrap
	rm -f bootstrap
