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
	cd go/src/operator && make crds
	cd helm/operator && make

build-images:
	cd helm/images-map && make


deploy-operator: build-docker build-operator-helm 
	cd out && helm upgrade --install operator operator.tgz -f images-manifest.yaml 

deploy-images: build-images
	cd out && helm upgrade --install images-map images-map.tgz -f images-manifest.yaml 


kill-dlv:
	killall operator || true
	killall dlv || true

dlv: kill-dlv
	cd go/src/operator && make compile
	./tool/run-dlv.sh
