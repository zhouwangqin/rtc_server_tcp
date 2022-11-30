FROM alpine:3.9.5

COPY ./bin/sfu /usr/bin/sfu
COPY ./configs/sfu.toml /etc/conf/sfu.toml

ENTRYPOINT ["/usr/bin/sfu", "-c", "/etc/conf/sfu.toml"]