FROM golang:1.10-alpine as builder

RUN apk add --update make

COPY . /go/src/github.com/inovex/mqtt-stresser
WORKDIR /go/src/github.com/inovex/mqtt-stresser
RUN make linux-static

FROM scratch
ARG BUILD_DATE=now
ARG VCS_REF=master
ARG VERSION=v1
LABEL org.label-schema.schema-version = "1.0"
LABEL org.label-schema.build-date=$BUILD_DATE
LABEL org.label-schema.name="mqtt-stresser"
LABEL org.label-schema.vcs-url="https://github.com/inovex/mqtt-stresser"
LABEL org.label-schema.vcs-ref=$VCS_REF
LABEL org.label-schema.version=$VERSION

COPY --from=builder /go/src/github.com/inovex/mqtt-stresser/build/mqtt-stresser.static /bin/mqtt-stresser
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT [ "/bin/mqtt-stresser" ]
