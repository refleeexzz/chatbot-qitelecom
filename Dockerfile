FROM golang:1.25-alpine AS build
WORKDIR /app
RUN apk add --no-cache gcc musl-dev sqlite-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
ENV PORT=8081
RUN go build -o qibot .
FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /app/qibot /app/qibot
COPY --from=build /app/index.html /app/index.html
COPY --from=build /app/bulk_messages.html /app/bulk_messages.html
COPY --from=build /app/admin_contacts.html /app/admin_contacts.html
COPY --from=build /app/admin_login.html /app/admin_login.html
COPY --from=build /app/admin_dashboard.html /app/admin_dashboard.html
EXPOSE 8081
ENTRYPOINT ["./qibot"]