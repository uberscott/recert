crds:
	operator-sdk generate crds
	operator-sdk generate k8s


compile: crds
	go build -o ../../bin/operator operator/cmd/manager
clean:
	rm  ../../bin/operator 

test:
	go test ./pkg/...

release: test compile


format:
	go fmt ./...


lint:
	golint ./...
