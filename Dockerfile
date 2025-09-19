FROM golang:1.25-alpine AS build
WORKDIR /app
RUN apk add --no-cache gcc musl-dev sqlite-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go build -o qibot .

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /app/qibot /app/qibot
COPY --from=build /app/index.html /app/index.html
EXPOSE 8081
ENTRYPOINT ["./qibot"]