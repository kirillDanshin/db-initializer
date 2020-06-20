FROM golang:latest as builder

COPY . /db-initializer

WORKDIR /db-initializer

RUN go build

FROM scratch

COPY --from=builder /db-initializer/db-initializer /db-initializer

ENTRYPOINT [ "/db-initializer" ]
