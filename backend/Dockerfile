FROM golang:1.27-alpine AS base
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

FROM base AS dev
EXPOSE 8080
CMD ["go", "run", "./cmd/api"]

FROM base AS builder
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22 AS prod
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /app/data /app/data
USER app
EXPOSE 8080
ENTRYPOINT ["/app/api"]
