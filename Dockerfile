FROM golang:1.26 AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /inventory-intel ./cmd/inventory-intel

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /inventory-intel /usr/local/bin/inventory-intel
COPY web ./web
COPY migrations ./migrations
RUN mkdir -p /app/data
EXPOSE 8080
CMD ["inventory-intel"]
