FROM golang:latest AS build

WORKDIR /app

ARG PROGRAM_NAME="discord-sync-bot"

# Install DBMate
RUN curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64
RUN chmod +x /usr/local/bin/dbmate

# Download project dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build Project
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./build/${PROGRAM_NAME}

# Move only necessary files into final image
FROM busybox
WORKDIR /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/local/bin/dbmate ./dbmate
COPY --from=build /app/db/migrations ./db/migrations
COPY --from=build /app/build/${PROGRAM_NAME} ./${PROGRAM_NAME}
EXPOSE 8080
CMD ["./discord-sync-bot"]
