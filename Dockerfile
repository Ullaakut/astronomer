FROM golang:1.12-alpine AS base-image

RUN apk --no-cache --no-progress add git ca-certificates && update-ca-certificates

ENV PROJECT_WORKING_DIR=
WORKDIR "/go/src/github.com/ullaakut/astronomer"
COPY . "/go/src/github.com/ullaakut/astronomer"


FROM base-image AS builder

ENV GO111MODULE=on
RUN mkdir -p /astronomer/etc/ssl/
COPY --from=base-image /etc/ssl/certs/ca-certificates.crt /astronomer/etc/ssl/certs/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /astronomer/astronomer *.go


FROM scratch AS base

COPY --from=builder /astronomer /

ENTRYPOINT ["/astronomer"]