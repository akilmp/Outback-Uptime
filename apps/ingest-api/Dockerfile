# Build stage
FROM golang:1.23.12 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ingest-api ./cmd/server

# Final image
FROM gcr.io/distroless/base-debian12
COPY --from=build /src/ingest-api /ingest-api
ENTRYPOINT ["/ingest-api"]

