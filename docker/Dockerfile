FROM alpine:3.2
RUN apk add --update ca-certificates # Certificates for SSL
ADD config2consul /go/bin/config2consul
ENTRYPOINT /go/bin/config2consul
