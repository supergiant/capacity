FROM golang AS build

# enable totally static binaries
ENV CGO_ENABLED "0"

# needed so we can mkdir in the scratch container later
RUN mkdir /tmp/emptydir

# get env2conf and a shell
RUN mkdir /tmp/bin
ADD https://github.com/supergiant/env2conf/releases/download/v1.0.0/env2conf /tmp/bin/
RUN chmod +x /tmp/bin/env2conf
ADD https://busybox.net/downloads/binaries/1.27.1-i686/busybox_ASH /tmp/bin/sh
RUN chmod +x /tmp/bin/sh


# build vendor stuff first to exploit cache
COPY vendor /go/src/
RUN cd /go/src && go install -v ./...

# do the build
WORKDIR /go
RUN mkdir -p src/github.com/supergiant/capacity
COPY . src/github.com/supergiant/capacity/
WORKDIR src/github.com/supergiant/capacity/cmd/capacity-service
RUN rm -Rf ../../vendor
RUN go build -v -ldflags="-s -w"

# add init script
COPY docker-init /tmp/bin/init
RUN chmod +x /tmp/bin/init

# build final container
FROM scratch
ENV PATH "/bin"
ENV SSL_CERT_FILE "/etc/ca-certificates.crt"
COPY --from=build /tmp/emptydir /etc
COPY --from=build /tmp/emptydir /etc/capacity-service
COPY --from=build /tmp/emptydir /incoming-config
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/
COPY --from=build /tmp/bin /bin
CMD ["init"]
