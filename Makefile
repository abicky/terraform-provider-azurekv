.PHONY: build
build:
	go build -o dist/

.PHONY: debug
debug:
	$(eval TEMPDIR := $(shell mktemp -d))
	go build -gcflags="all=-N -l" -o $(TEMPDIR)
	dlv exec --listen=:2345 --accept-multiclient --continue --headless $(TEMPDIR)/terraform-provider-azurekv -- -debug

.PHONY: generate
generate:
	cd tools; go generate ./...

.PHONY: fmt
fmt:
	gofmt -s -w -e .

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v -cover -timeout 120s -parallel 10 ./...

.PHONY: testacc
testacc:
	TF_ACC=1 go test -v -parallel 32 -cover -timeout 120m ./... $(TESTARGS)
