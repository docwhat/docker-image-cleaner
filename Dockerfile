FROM golang:alpine
MAINTAINER Christian Höltje <docwhat@gerf.org>

RUN apk add --update --no-cache go git

COPY .git *.go /go/src/github.com/docwhat/docker-image-cleaner/
RUN go get github.com/docwhat/docker-image-cleaner

RUN apk del go git

ENTRYPOINT ["/go/bin/docker-image-cleaner"]
