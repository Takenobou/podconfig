    FROM golang:1.24-alpine AS builder

    WORKDIR /app
    
    COPY go.mod go.sum ./
    RUN go mod download
    
    COPY . .
    
    RUN go build -o /podconfig ./cmd/podconfig
    
    FROM alpine:latest
    
    RUN apk --no-cache add ca-certificates
    
    WORKDIR /app
    COPY --from=builder /podconfig /app/podconfig
    
    EXPOSE 8080
    
    ENTRYPOINT ["/app/podconfig"]