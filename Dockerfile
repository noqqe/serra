FROM golang:1.20

WORKDIR /go/src/app
COPY src /go/src/app/src
COPY templates /go/src/app/templates
COPY go.mod /go/src/app/go.mod
COPY go.sum /go/src/app/go.sum
COPY .git /go/src/app/.git
COPY serra.go /go/src/app/serra.go


RUN go get -v ./...
RUN go build -ldflags "-X github.com/noqqe/serra/src/serra.Version=`git describe --tags`"  -v serra.go

# Run radsportsalat
EXPOSE 8080
CMD [ "./serra", "web" ]
