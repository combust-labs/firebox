FROM golang:1.16-alpine3.13 as builder

RUN apk add alpine-sdk ca-certificates

WORKDIR "/code"
COPY . "/code"
RUN make clean vendor build

FROM alpine:3.13
COPY --from=builder /code/bin/firebox /firebox
ENTRYPOINT ["/firebox"]
