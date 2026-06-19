# Multi-stage build for the cofiswarm-agent-registry sidecar.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./cmd/cofiswarm-agent-registry

FROM alpine:3.20
RUN adduser -D -u 10001 app
COPY --from=build /out/app /usr/local/bin/cofiswarm-agent-registry
USER app
EXPOSE 8012
ENTRYPOINT ["/usr/local/bin/cofiswarm-agent-registry"]
