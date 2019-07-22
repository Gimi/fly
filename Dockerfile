FROM alpine:3.7
LABEL maintainer="Gimi Liang"

ARG UID=998
ARG GID=998
RUN apk update \
 && apk add --no-cache ca-certificates \
 && wget -O /usr/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.1/dumb-init_1.2.1_amd64 \
 && chmod +x /usr/bin/dumb-init \
 && rm -rf /var/cache/apk/* \
 && rm -rf /tmp/* /var/tmp/* \
 && addgroup -S -g $GID gimi \
 && adduser -D -S -H -u $UID -G gimi gimi
USER $UID:$GID
COPY --chown=gimi:gimi ./build/hi entrypoint.sh /usr/bin/
RUN chmod +x /usr/bin/entrypoint.sh /usr/bin/hi 
ENTRYPOINT ["/usr/bin/entrypoint.sh"]
