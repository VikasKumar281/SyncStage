FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN go vet ./... && go test ./...


RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/server ./cmd/server

FROM alpine:3.20


RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 sequencer

WORKDIR /app

COPY --from=build /out/server /app/server

RUN mkdir -p /app/data && chown -R sequencer:sequencer /app
VOLUME ["/app/data"]

USER sequencer

ENV PORT=8080 \
    DATA_PATH=/app/data/state.json \
    ALLOWED_ORIGINS=*

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT}/api/health" || exit 1

ENTRYPOINT ["/app/server"]
