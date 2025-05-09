FROM golang:1.24 AS builder

WORKDIR /app

ENV GOARCH=amd64

ENV GOOS=linux

COPY src/go.mod src/go.sum ./

RUN go mod download

COPY src/ .

RUN go build -o /app/main .

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add libc6-compat

COPY --from=builder /app/main .

COPY src/recipes.json .

RUN chmod +x /app/main

EXPOSE 8080

CMD ["./main"]
