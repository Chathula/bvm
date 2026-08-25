.PHONY: build test test-e2e vet fmt tidy clean

build:
	go build -o bin/ .

test:
	go test ./...

test-e2e:
	go test -tags=e2e ./e2e -v

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

clean:
	rm -rf bin/
