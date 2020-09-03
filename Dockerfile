# Build Geth in a stock Go builder container
FROM golang:1.10-alpine as construction

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /ups
RUN cd /ups && make gups

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=construction /ups/build/bin/gups /usr/local/bin/
CMD ["gups"]

EXPOSE 8545 8545 9215 9215 30310 30310 30311 30311 30313 30313
ENTRYPOINT ["gups"]


