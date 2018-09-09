FROM golang AS build
ENV CGO_ENABLED "0"
RUN apt update
RUN apt install upx-ucl -y
WORKDIR /go
RUN mkdir /tmp/bin
RUN mkdir /tmp/emptydir
ADD https://github.com/supergiant/env2conf/releases/download/v1.0.0/env2conf /tmp/bin/
RUN chmod +x /tmp/bin/env2conf
ADD https://busybox.net/downloads/binaries/1.27.1-i686/busybox_ASH /tmp/bin/sh
RUN chmod +x /tmp/bin/sh
RUN upx --brute /tmp/bin/sh



COPY vendor /go/src/
RUN cd /go/src && go install -v ./...

RUN mkdir -p src/github.com/supergiant/capacity
COPY . src/github.com/supergiant/capacity/
WORKDIR src/github.com/supergiant/capacity/cmd/capacity-service
RUN rm -Rf ../../vendor



RUN go build -v -ldflags="-s -w"
RUN ls -lh capacity-service
#RUN upx --brute capacity-service
RUN mv capacity-service /tmp/bin/
ADD docker-init /tmp/bin/init
RUN chmod +x /tmp/bin/init
RUN ls -lh /tmp/bin


FROM scratch
ENV PATH "/bin"
ENV SSL_CERT_FILE "/etc/ca-certificates.crt"
COPY --from=build /tmp/emptydir /etc
COPY --from=build /tmp/emptydir /etc/capacity-service
COPY --from=build /tmp/emptydir /incoming-config
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/
COPY --from=build /tmp/bin /bin
CMD ["init"]
