FROM golang:stretch AS build

# install dependencies (npm isn't avalible in stretch)
RUN sed -i -e 's/stretch/buster/g' /etc/apt/sources.list
RUN apt-get update && apt-get install -y npm build-essential
RUN npm i npm@latest -g

# enable totally static binaries
ENV CGO_ENABLED "0"

# needed so we can mkdir in the scratch container later
RUN mkdir /tmp/emptydir

# get env2conf and a shell
RUN mkdir /tmp/bin
ADD https://github.com/supergiant/env2conf/releases/download/v1.1.0/env2conf /tmp/bin/
RUN chmod +x /tmp/bin/env2conf
ADD https://busybox.net/downloads/binaries/1.27.1-i686/busybox_ASH /tmp/bin/sh
RUN chmod +x /tmp/bin/sh

# build vendor stuff first to exploit cache
COPY vendor /go/src/
RUN cd /go/src && go install -v ./...

# build the UI
COPY cmd/capacity-service/ui/capacity-service /tmp/ui
WORKDIR /tmp/ui
RUN npm install
RUN npm install -g @angular/cli
RUN ng build --prod --base-href="/ui/"

# do the build
RUN mkdir -p /go/src/github.com/supergiant/capacity
COPY . /go/src/github.com/supergiant/capacity/
WORKDIR /go/src/github.com/supergiant/capacity/cmd/capacity-service
RUN rm -Rf ../../vendor
# Put pre-built ui back in place
RUN rm -Rf ui/capacity-service
RUN mv /tmp/ui ui/capacity-service
RUN packr build -v -ldflags="-s -w"
RUN mv capacity-service /tmp/bin/

# add init script
COPY docker-init /tmp/bin/init
RUN chmod +x /tmp/bin/init

# build final container
FROM scratch
ENV PATH "/bin"
ENV SSL_CERT_FILE "/etc/ca-certificates.crt"
COPY --from=build /tmp/emptydir /etc
COPY --from=build /tmp/emptydir /etc/capacity-service
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/
COPY --from=build /tmp/bin /bin
CMD ["init"]
