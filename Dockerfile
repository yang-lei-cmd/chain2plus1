# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/main.go

# ============================================================
# Stage 2: Runtime
# ============================================================
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
