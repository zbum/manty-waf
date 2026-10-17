FROM alpine:3.21
COPY dist/manty-waf-linux /usr/local/bin/manty-waf
COPY dist/echoapp-linux /usr/local/bin/echoapp
USER nobody
ENTRYPOINT ["/usr/local/bin/manty-waf"]
