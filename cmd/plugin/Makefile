.PHONY: all
all:
	$(MAKE) GOOS=linux GOARCH=arm64 osarch-build
	$(MAKE) GOOS=darwin GOARCH=arm64 osarch-build
	$(MAKE) GOOS=linux GOARCH=amd64 osarch-build
	$(MAKE) GOOS=darwin GOARCH=amd64 osarch-build
	$(MAKE) GOOS=windows GOARCH=amd64 osarch-build

.PHONY: clean
clean:
	$(MAKE) GOOS=linux GOARCH=arm64 osarch-clean
	$(MAKE) GOOS=darwin GOARCH=arm64 osarch-clean
	$(MAKE) GOOS=linux GOARCH=amd64 osarch-clean
	$(MAKE) GOOS=darwin GOARCH=amd64 osarch-clean
	$(MAKE) GOOS=windows GOARCH=amd64 osarch-clean

.PHONY: osarch-build
osarch-build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o iww-plugin-$(GOOS)-$(GOARCH)

.PHONY: osarch-clean
osarch-clean:
	GOOS=$(GOOS) GOARCH=$(GOARCH) rm -f iww-plugin-$(GOOS)-$(GOARCH)

# make and install the iww plugin on a mac
.PHONY: mac
mac: clean
	$(MAKE) GOOS=darwin GOARCH=arm64 osarch-build
	ibmcloud plugin install -f iww-plugin-darwin-arm64-*
