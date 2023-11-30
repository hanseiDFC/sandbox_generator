FROM golang:alpine AS builder

RUN mkdir /app

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod tidy

COPY . .

RUN go build -o server .

FROM alpine

COPY --from=builder /app/server /app
COPY --from=builder /app/challenges.json /challenges.json
COPY --from=builder /app/templates /templates

EXPOSE 5000

CMD /app