FROM golang:1.12-alpine AS base-image

RUN apk --no-cache --no-progress add git ca-certificates && update-ca-certificates

FROM scratch AS base

COPY --from=base-image /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY dist/astronomer-linux-amd64 /astronomer

ENTRYPOINT ["/astronomer"]