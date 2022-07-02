.PHONY: all
all:
	$(MAKE) GOOS=linux GOARCH=amd64 osarch-build
	$(MAKE) GOOS=darwin GOARCH=amd64 osarch-build
	$(MAKE) GOOS=windows GOARCH=amd64 osarch-build

.PHONY: clean
clean:
	$(MAKE) GOOS=linux GOARCH=amd64 osarch-clean
	$(MAKE) GOOS=darwin GOARCH=amd64 osarch-clean
	$(MAKE) GOOS=windows GOARCH=amd64 osarch-clean

.PHONY: osarch-build
osarch-build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o iww-$(GOOS)-$(GOARCH)

.PHONY: osarch-clean
osarch-clean:
	GOOS=$(GOOS) GOARCH=$(GOARCH) rm -f iww-$(GOOS)-$(GOARCH)

mac:
	$(MAKE) GOOS=darwin GOARCH=amd64 osarch-build
