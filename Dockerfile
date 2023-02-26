FROM golang:1.20

WORKDIR /go/src/app
COPY . /go/src/app

RUN go get -v ./...
RUN go build -ldflags "-X github.com/noqqe/serra/src/serra.Version=`git describe --tags`"  -v serra.go

# Run radsportsalat
EXPOSE 8080
CMD [ "./serra", "web" ]
