FROM golang:alpine AS builder

RUN mkdir /app

WORKDIR /app

COPY . .

RUN go build -o server .

FROM alpine

COPY --from=builder /app/server /app
COPY --from=builder /app/challenges.json /challenges.json

EXPOSE 5000

CMD /app