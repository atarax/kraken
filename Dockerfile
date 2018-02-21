FROM alpine:latest as certificates
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# RUNTIME

FROM scratch
COPY --from=certificates /etc/ssl/certs /etc/ssl/certs
COPY bin/kraken /usr/bin/kraken
CMD ["/usr/bin/kraken"]