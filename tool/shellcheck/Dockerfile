FROM ubuntu:latest

RUN apt-get update && apt-get -y install shellcheck

VOLUME /work

WORKDIR /work

ENTRYPOINT shellcheck
