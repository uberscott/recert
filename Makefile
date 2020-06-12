init:
	./tool/up2date.sh 
	mkdir out || true 

all: test format build build-edb-operator-helm

build: init build-docker build-operator-helm

build-docker: init
	skaffold build --file-output=out/images-manifest.json
	cat out/images-manifest.json | docker run -i --rm  edb-tool_skaffold-manifest-converter > out/images-manifest.yaml

test: init
	./bash/loop.sh test

format: init
	cd go && make format

pull: init
	for v in v9.6 v10 v11 v12; do for tag in edb-as edb-pg edb-as-gis; do docker pull containers-dt.enterprisedb.com/test/$tag:$v; done; done

tools:
	./tool/build.sh

clean:
	rm -rf out

build-operator-helm:
	cd helm/edb-operator && make

deploy-operator: build-docker build-operator-helm 
	cd out && helm upgrade --install edb-operator edb-operator.tgz -f images-manifest.yaml -n operators
