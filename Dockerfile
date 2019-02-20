FROM golang:alpine as builder

WORKDIR /build

ADD server.go /build/server.go

RUN apk add git && \
    go get -d



RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main server.go

FROM alpine
MAINTAINER Andreas Peters <support@aventer.biz>

ENV IMAPSERVER "imag.gmail.com"
ENV IMAPPORT "143"
ENV CALLBACKURL "http://localhost:9094"
ENV CLIENTID "1"
ENV CLIENTSECRET "2"
ENV DOMAIN "gmail.com"

RUN adduser -S -D -H -h /app appuser

USER appuser

COPY --from=builder /build/main /app/

COPY static /app/static

EXPOSE 9094

WORKDIR "/app"

CMD ["./main"]