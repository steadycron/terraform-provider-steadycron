default: build

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test ./... -v -count=1 -timeout 120s

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./internal/provider/... -v -count=1 -timeout 120m

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: docs
docs:
	go generate ./...

.PHONY: install
install: build
	go install .

.PHONY: clean
clean:
	rm -rf dist/

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: vet
vet:
	go vet ./...
