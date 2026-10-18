# Этап 1: сборка 
FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/crm ./cmd/crm

# Этап 2: запуск
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/crm /crm

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/crm"]