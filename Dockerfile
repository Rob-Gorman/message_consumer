# Build stage
FROM golang:1.18-alpine AS build

WORKDIR /app
COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -o /delivery-agent ./cmd

## Deploy
FROM scratch

COPY --from=build /delivery-agent /delivery-agent

ENTRYPOINT ["/delivery-agent"]
