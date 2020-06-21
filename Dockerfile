FROM golang:latest as builder

COPY . /db-initializer

WORKDIR /db-initializer

RUN go build

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /db-initializer/db-initializer /db-initializer

ENTRYPOINT [ "/db-initializer" ]
