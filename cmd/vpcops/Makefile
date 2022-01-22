doit:
	go build ./...
	./vpcops > vpc_operations.go
	go fmt vpc_operations.go
	mv vpc_operations.go vpc_operations.go.generated
	@if diff vpc_operations.go.generated ../../iww/vpc_operations.go; then \
		echo no changes; \
  else \
		echo replace:; \
		echo mv vpc_operations.go.generated ../../iww/vpc_operations.go; \
	fi	
