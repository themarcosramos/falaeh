FROM golang:1.27-alpine AS base
WORKDIR /app
COPY go.mod ./
RUN for attempt in 1 2 3 4 5; do \
        if go mod download; then exit 0; fi; \
        if [ "$attempt" -eq 5 ]; then \
            echo "go mod download failed after $attempt attempts" >&2; \
            exit 1; \
        fi; \
        delay=$((attempt * 5)); \
        echo "go mod download failed; retrying in ${delay}s (attempt $((attempt + 1))/5)" >&2; \
        sleep "$delay"; \
    done
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
