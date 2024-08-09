FROM golang:latest AS build

WORKDIR /app

# Download project dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build Project
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /build-exe

# Move only necessary files into final image
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build-exe /build-exe
CMD ["/build-exe"]
