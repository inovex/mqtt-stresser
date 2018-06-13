FROM golang:1.10-alpine as builder

RUN apk add --update make

COPY . /go/src/github.com/inovex/mqtt-stresser
WORKDIR /go/src/github.com/inovex/mqtt-stresser
RUN make linux-static

FROM scratch

COPY --from=builder /go/src/github.com/inovex/mqtt-stresser/build/mqtt-stresser.static /bin/mqtt-stresser
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT [ "/bin/mqtt-stresser" ]
