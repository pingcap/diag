FROM alpine:3.15

RUN apk add tzdata --no-cache
ADD bin/k8s-pod /usr/local/bin/diag
ADD bin/info.toml /usr/local/bin/info.toml
RUN mkdir -p /diag/package /diag/collector
RUN chmod 755 -R /diag

