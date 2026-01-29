FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/migrate_apply ./cmd/migrate_apply

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/app /bin/app
COPY --from=build /bin/migrate_apply /bin/migrate_apply
COPY --from=build /app/internal/migrations /app/internal/migrations
COPY entrypoint.sh /bin/entrypoint.sh
RUN chmod +x /bin/entrypoint.sh
WORKDIR /app
ENV APP_PORT=8080
EXPOSE 8080
CMD ["/bin/entrypoint.sh"]
