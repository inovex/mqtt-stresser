FROM golang:1.12-alpine as builder

RUN apk add --update make

COPY . /go/src/github.com/inovex/mqtt-stresser
WORKDIR /go/src/github.com/inovex/mqtt-stresser
RUN make linux-static-amd64

FROM scratch
ARG BUILD_DATE="1985-04-12T23:20:50.52Z"
ARG VCS_REF=not-specified
ARG VERSION=not-specified
LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.build-date=$BUILD_DATE
LABEL org.label-schema.name="mqtt-stresser"
LABEL org.label-schema.vcs-url="https://github.com/inovex/mqtt-stresser"
LABEL org.label-schema.vcs-ref=$VCS_REF
LABEL org.label-schema.version=$VERSION

COPY --from=builder /go/src/github.com/inovex/mqtt-stresser/build/mqtt-stresser-linux-amd64-static /bin/mqtt-stresser
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT [ "/bin/mqtt-stresser" ]
