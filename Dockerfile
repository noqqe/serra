FROM golang:1.25-alpine AS build

RUN apk update && apk add --no-cache git ca-certificates curl

WORKDIR /go/src/app
COPY pkg /go/src/app/pkg
COPY cmd /go/src/app/cmd
COPY templates /go/src/app/templates
COPY go.mod /go/src/app/go.mod
COPY go.sum /go/src/app/go.sum
COPY .git /go/src/app/.git

# build
RUN go build -ldflags "-X github.com/noqqe/serra/pkg/serra.Version=`git describe --tags`"  -v cmd/serra/serra.go

# copy
FROM scratch
WORKDIR /go/src/app
COPY --from=build /go/src/app/serra /go/src/app/serra
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /tmp /tmp
COPY templates /go/src/app/templates

# run
EXPOSE 8080
CMD [ "/go/src/app/serra", "web" ]
