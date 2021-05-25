# Build Gpop in a stock Go builder container
FROM golang:1.16-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-popcateum
RUN cd /go-popcateum && make gpop

# Pull Gpop into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-popcateum/build/bin/gpop /usr/local/bin/

EXPOSE 9506 9507 60606 60606/udp
ENTRYPOINT ["gpop"]
