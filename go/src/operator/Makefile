crds:
	operator-sdk generate crds
	#openapi-gen --logtostderr=true -o "" -i ./pkg/apis/enterprisedb/v1alpha1 -O zz_generated.openapi -p ./pkg/apis/enterprisedb/v1alpha1 -h ./boilerplate.go.txt -r "-"
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
