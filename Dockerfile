FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /go/src/app
COPY src /go/src/app/src
COPY templates /go/src/app/templates
COPY go.mod /go/src/app/go.mod
COPY go.sum /go/src/app/go.sum
COPY .git /go/src/app/.git
COPY serra.go /go/src/app/serra.go

# build
RUN go get -v ./...
RUN go build -ldflags "-X github.com/noqqe/serra/src/serra.Version=`git describe --tags`"  -v serra.go

# copy
FROM scratch
WORKDIR /go/src/app
COPY --from=builder /go/src/app/serra /go/src/app/serra
COPY templates /go/src/app/templates

# run
EXPOSE 8080
CMD [ "/go/src/app/serra", "web" ]
