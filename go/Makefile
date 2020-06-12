all: test format build

build:
	../bash//loop.sh build
test:
	../bash//loop.sh test
format:
	gofmt -w ./src


clean:
	chmod -R 744 pkg
	rm -rf bin pkg
