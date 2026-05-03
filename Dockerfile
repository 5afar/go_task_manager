FROM golang:1.20-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/pet ./cmd/server

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /app/pet /usr/local/bin/pet
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/pet"]
