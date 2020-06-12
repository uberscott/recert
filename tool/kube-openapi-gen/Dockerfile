FROM golang:latest AS builder

RUN apt-get update

RUN mkdir -p $GOPATH/src/openapi && \
    cd $GOPATH/src/openapi && \
    go mod init enterprisedb.com/kube-openapi-gen && \
    go get k8s.io/kube-openapi && \
    go build -o /usr/bin/kube-openapi-gen k8s.io/kube-openapi/cmd/openapi-gen
  

FROM ubuntu:latest

COPY --from=builder /usr/bin/kube-openapi-gen /usr/bin/kube-openapi-gen 

RUN apt-get -y update 

ADD tool/kube-openapi-gen/script.sh /opt/script.sh
ADD tool/kube-openapi-gen/boilerplate.go.txt /opt/

VOLUME /work

WORKDIR /work

ENTRYPOINT /opt/script.sh
