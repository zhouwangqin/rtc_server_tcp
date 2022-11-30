FROM alpine:3.9.5

COPY ./bin/islb /usr/bin/islb
COPY ./configs/islb.toml /etc/conf/islb.toml

ENTRYPOINT ["/usr/bin/islb", "-c", "/etc/conf/islb.toml"]