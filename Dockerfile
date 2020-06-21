FROM golang:latest as builder

COPY . /db-initializer

WORKDIR /db-initializer

RUN CGO_ENABLED=0 go build

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /db-initializer/db-initializer /db-initializer

ENTRYPOINT [ "/db-initializer" ]
