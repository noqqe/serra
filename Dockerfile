FROM golang:1.21-alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /go/src/app
COPY pkg /go/src/app/pkg
COPY cmd /go/src/app/cmd
COPY templates /go/src/app/templates
COPY go.mod /go/src/app/go.mod
COPY go.sum /go/src/app/go.sum
COPY .git /go/src/app/.git

# build
RUN go build -ldflags "-X github.com/noqqe/serra/src/serra.Version=`git describe --tags`"  -v cmd/serra/serra.go

# copy
FROM scratch
WORKDIR /go/src/app
COPY --from=builder /go/src/app/serra /go/src/app/serra
COPY templates /go/src/app/templates

# run
EXPOSE 8080
CMD [ "/go/src/app/serra", "web" ]
