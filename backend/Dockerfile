FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
# -p 2 caps compile parallelism so the build fits small VPS / constrained hosts.
RUN CGO_ENABLED=0 go build -p 2 -o /out/api ./cmd/server
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/api /api
USER nonroot:nonroot
ENTRYPOINT ["/api"]

