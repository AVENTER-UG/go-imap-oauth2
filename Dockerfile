FROM golang:alpine AS builder

WORKDIR /build

COPY . /build/

RUN apk add git && \
    go get -d



RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .

FROM alpine
LABEL maintainer="Andreas Peters <support@aventer.biz>"
LABEL org.opencontainers.image.title="go-imap-oauth2"
LABEL org.opencontainers.image.description="Simpe oauth2 provider for imap"
LABEL org.opencontainers.image.vendor="AVENTER UG (haftungsbeschränkt)"
LABEL org.opencontainers.image.source="https://github.com/AVENTER-UG/"

ENV IMAPSERVER="imag.gmail.com"
ENV IMAPPORT="143"
ENV CALLBACKURL="http://localhost:9094"
ENV CLIENTID="1"
ENV CLIENTSECRET="2"
ENV DOMAIN="gmail.com"

RUN adduser -S -D -H -h /app appuser

USER appuser

COPY --from=builder /build/main /app/

COPY static /app/static

EXPOSE 9094

WORKDIR "/app"

CMD ["./main"]
