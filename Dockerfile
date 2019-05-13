FROM alpine:latest

ADD lcow-injector /lcow-injector
ENTRYPOINT ["./lcow-injector"]