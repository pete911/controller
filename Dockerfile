FROM golang:1.21.3-alpine AS build
WORKDIR /go/src/app

COPY . .
RUN go build -mod vendor -o /bin/controller


FROM alpine:3.18.4
MAINTAINER Peter Reisinger <p.reisinger@gmail.com>

COPY --from=build /bin/controller /usr/local/bin/controller
CMD ["controller"]
