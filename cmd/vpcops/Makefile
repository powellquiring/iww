doit:
	go build main.go
	./main > vpc_operations.go
	go fmt vpc_operations.go
	@if ! diff vpc_operations.go ../../iww; then \
		echo mv vpc_operations.go ../../iww; \
	fi	
