FROM golang:1.25 as build

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/govein main.go

FROM alpine:latest as run

WORKDIR /app
COPY --from=build /app/govein /app/govein

ENTRYPOINT ["/app/govein"]