FROM docker.io/library/golang:1.25.0 AS builder
COPY . /src
WORKDIR /src
ENV CGO_ENABLED=0
RUN go build -o dockyards-pdns -ldflags="-s -w"

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /src/dockyards-pdns /usr/bin/dockyards-pdns
ENTRYPOINT ["/usr/bin/dockyards-pdns"]
