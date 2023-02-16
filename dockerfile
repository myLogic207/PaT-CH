# syntax=docker/dockerfile:1
FROM golang:1.20.1

WORKDIR /app
 
COPY src/ ./
 
RUN go build -o bin/patch
EXPOSE 8080
CMD ["bin/patch"]
