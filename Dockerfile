# syntax=docker/dockerfile:1

FROM golang:1.23.5-alpine as builder
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /darkstat-exporter

FROM scratch
COPY --from=builder /darkstat-exporter /darkstat-exporter
EXPOSE 9090
WORKDIR /
CMD [ "/darkstat-exporter" ]
