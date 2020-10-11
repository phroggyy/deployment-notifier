FROM golang:1.13 as builder

WORKDIR /go/src/app

ADD go.mod .
ADD go.sum .

RUN go mod download
RUN go mod verify

ADD . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /go/bin/deploy-notifier

FROM gcr.io/distroless/base-debian10:nonroot

COPY --from=builder /go/bin/deploy-notifier /go/bin/deploy-notifier
USER nonroot:nonroot

ENTRYPOINT ["/go/bin/deploy-notifier"]
