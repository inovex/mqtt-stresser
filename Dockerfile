FROM golang:1.10-alpine

RUN apk add --update make git zip

ENV GOPATH /usr/local/go

RUN mkdir -p ${GOPATH}/src/github.com/inovex/ && \
    git clone https://github.com/inovex/mqtt-stresser.git ${GOPATH}/src/github.com/inovex/mqtt-stresser/

RUN cd ${GOPATH}/src/github.com/inovex/mqtt-stresser/ && \
    make

RUN cd ${GOPATH}/src/github.com/inovex/mqtt-stresser/build && \
    tar -xzvf mqtt-stresser-linux-amd64.tar.gz && \
    cp mqtt-stresser /bin

ENTRYPOINT [ "/bin/mqtt-stresser" ]
