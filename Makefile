.PHONY: build test test-e2e vet fmt tidy clean cover

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

# Reports coverage and fails unless every function is at least 90% covered.
# The main() entrypoint is excluded: it only runs inside the built binary,
# so it cannot execute under `go test`.
cover:
	go test ./command/ ./util/ -coverprofile=coverage.out
	go tool cover -func=coverage.out | tee coverage.txt | tail -1
	@awk '$$3+0 < 90 && $$2 != "main" { print "UNDER 90%:", $$1, $$3; bad = 1 } END { if (bad) exit 1 }' coverage.txt
