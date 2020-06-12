FROM golang:latest 

RUN go get -u golang.org/x/lint/golint

ENV PATH=$PATH:$GOPATH/bin

VOLUME /work

WORKDIR /work

ENTRYPOINT golint


