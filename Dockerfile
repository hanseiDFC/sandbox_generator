FROM golang:alpine AS builder

RUN mkdir /app

WORKDIR /app

COPY . .

RUN go mod tidy

RUN go build -o server .

FROM alpine

COPY --from=builder /app/server /app
COPY --from=builder /app/challenges.json /challenges.json
COPY --from=builder /app/templates /templates

EXPOSE 5000

CMD /app