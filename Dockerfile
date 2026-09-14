# syntax=docker/dockerfile:1.7
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/settlementctl ./cmd/settlementctl

FROM alpine:3.24
RUN addgroup -S app && adduser -S -G app -u 10001 app
COPY --from=build /out/settlementctl /usr/local/bin/settlementctl
USER app
WORKDIR /workspace
ENTRYPOINT ["settlementctl"]
CMD ["help"]

