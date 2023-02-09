FROM alpine:latest

RUN set -eux; \
    apk add ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

WORKDIR /app

COPY main /app/
COPY templates /app/templates

CMD ["./main"]
