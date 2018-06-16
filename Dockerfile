FROM golang:1.9.7-alpine
MAINTAINER Andreas Peters <support@aventer.biz>

ENV IMAPSERVER "imag.gmail.com"
ENV IMAPPORT "143"
ENV CALLBACKURL "http://localhost:9094"
ENV CLIENTID "1"
ENV CLIENTSECRET "2"
ENV DOMAIN "gmail.com"


RUN apk update; apk add git && \
    go get github.com/wxdao/go-imap/imap && \
    go get gopkg.in/oauth2.v3/... && \
    mkdir /app 

COPY static /app/static
ADD server.go /app/server.go
ADD start.sh /app/start.sh

EXPOSE 9094

WORKDIR "/app"

ENTRYPOINT ["/app/start.sh"]
