FROM golang:1.27.3-alpine

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./cmd/api

ARG SERVER_PORT
EXPOSE  ${SERVER_PORT}

CMD ["/main"]