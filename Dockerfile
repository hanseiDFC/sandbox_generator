FROM golang:1.21.4-alpine AS builder

RUN mkdir /app

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY *.go .

RUN go build -o server .

FROM alpine:3.18.4

COPY --from=builder /app/server /app
COPY challenges.json .
COPY templates /templates

EXPOSE 8000

CMD /app