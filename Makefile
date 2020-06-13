init:
	./tool/up2date.sh 
	mkdir out || true 

all: test format build build-operator-helm

build: init build-docker build-operator-helm

build-docker: init
	skaffold build --file-output=out/images-manifest.json
	cat out/images-manifest.json | docker run -i --rm  tool_skaffold-manifest-converter > out/images-manifest.yaml

test: init
	./bash/loop.sh test

format: init
	cd go && make format

tools:
	./tool/build.sh

clean:
	rm -rf out

build-operator-helm:
	cd helm/operator && make

deploy-operator: build-docker build-operator-helm 
	cd out && helm upgrade --install operator operator.tgz -f images-manifest.yaml -n recert 
