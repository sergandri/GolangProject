FROM golang:1.22-alpine
LABEL authors="serga"
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o /transactions_server
EXPOSE 8080
CMD ["/transactions_server"]
