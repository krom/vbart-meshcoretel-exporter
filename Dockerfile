# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /out/vbart-meshcoretel-exporter .

FROM gcr.io/distroless/static-debian12:nonroot AS final

COPY --from=builder /out/vbart-meshcoretel-exporter /usr/bin/vbart-meshcoretel-exporter

USER nonroot:nonroot

EXPOSE 9642

ENTRYPOINT ["/usr/bin/vbart-meshcoretel-exporter"]
