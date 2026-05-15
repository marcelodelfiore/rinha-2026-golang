FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/rinha-api ./cmd/api

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/rinha-api /bin/rinha-api
COPY resources ./resources
COPY resources/references_u8.bin resources/references_u8.bin

ENV PORT=8080

CMD ["/bin/rinha-api"]
