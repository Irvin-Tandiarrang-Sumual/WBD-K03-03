FROM golang:1.25.5-alpine

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /migrate ./cmd/migrate && \
    CGO_ENABLED=0 GOOS=linux go build -o /seed ./cmd/seed && \
    go clean -cache

ARG SERVER_PORT
EXPOSE  ${SERVER_PORT}

CMD ["sh", "-c", "if [ \"$SEED_TEST_DATA\" = \"true\" ]; then /seed; else /migrate; fi && exec /main"]
