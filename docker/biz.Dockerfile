FROM alpine:3.9.5

COPY ./bin/biz /usr/bin/biz
COPY ./configs/biz.toml /etc/conf/biz.toml

ENTRYPOINT ["/usr/bin/biz", "-c", "/etc/conf/biz.toml"]

