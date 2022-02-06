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
	MAJOR=$$(cat iww.go | grep Major | awk '{print $$2}' | tr -d ,); \
	MINOR=$$(cat iww.go | grep Minor | awk '{print $$2}' | tr -d ,); \
	BUILD=$$(cat iww.go | grep Build | awk '{print $$2}' | tr -d ,); \
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o iww-plugin-$(GOOS)-$(GOARCH)-$$MAJOR.$$MINOR.$$BUILD

.PHONY: osarch-clean
osarch-clean:
	GOOS=$(GOOS) GOARCH=$(GOARCH) rm -f iww-plugin-$(GOOS)-$(GOARCH)-*

plugin:
	$(MAKE) GOOS=darwin GOARCH=amd64 osarch-build
	ibmcloud plugin install -f iww-plugin-darwin-amd64
