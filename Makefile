macli:
	cd iww; make
	cd cmd/iww; make mac

mac:
	cd iww; make
	cd cmd/iww; make mac
	cd cmd/plugin; make mac

all:
	cd iww; make
	cd cmd/iww; make; ls -l iww-*
	cd cmd/plugin; make;  ls -l iww-*

tag:
	MAJOR=$$(cat cmd/plugin/iww.go | grep Major | awk '{print $$2}' | tr -d ,); \
	MINOR=$$(cat cmd/plugin/iww.go | grep Minor | awk '{print $$2}' | tr -d ,); \
	BUILD=$$(cat cmd/plugin/iww.go | grep Build | awk '{print $$2}' | tr -d ,); \
	git tag $$MAJOR.$$MINOR.$$BUILD; \
	git push origin $$MAJOR.$$MINOR.$$BUILD