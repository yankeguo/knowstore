FROM golang:1.23 AS builder

ENV CGO_ENABLED="0"

WORKDIR /go/src/app

ADD . .

RUN go build -o /knowstore .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /knowstore /knowstore

CMD ["/knowstore"]