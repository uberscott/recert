FROM ubuntu:latest

RUN apt-get -y update && \
    apt-get -y install jq python3-pip

RUN pip3 install yq


ADD tool/skaffold-manifest-converter/script.sh /opt/script.sh


CMD /opt/script.sh

