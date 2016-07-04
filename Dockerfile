FROM golang:1.6.2-alpine

RUN apk add --update git && apk add --update make && rm -rf /var/cache/apk/*

ADD . /go/src/github.com/ernestio/nat-builder
WORKDIR /go/src/github.com/ernestio/nat-builder

RUN make deps && go install

ENTRYPOINT /go/bin/nat-builder