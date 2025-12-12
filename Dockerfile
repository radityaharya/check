# Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Copy root-level pnpm workspace files
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY web/package.json ./web/

RUN corepack enable && pnpm install --frozen-lockfile

COPY web/ ./web/
RUN pnpm --filter web build

# Build backend
FROM golang:1.25.5-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o gocheck .

# Final image
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /app/gocheck .
COPY --from=frontend-builder /app/web/dist ./web/dist

RUN mkdir -p /data

EXPOSE 8080

CMD ["./gocheck"]

