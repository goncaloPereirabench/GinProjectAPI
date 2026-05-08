FROM golang:1.26-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=build /out/api /app/api

USER appuser
EXPOSE 8080

ENTRYPOINT ["/app/api"]
