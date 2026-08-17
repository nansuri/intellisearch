FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
# -p 2 caps compile parallelism so the build fits small VPS / constrained hosts.
RUN CGO_ENABLED=0 go build -p 2 -o /out/api ./cmd/server
# Pre-create the uploads root (with subdirs) in the build stage so the final
# unprivileged image can own it. The runtime runs as nonroot on distroless,
# where the filesystem root is read-only — without a pre-created, owned
# /uploads, every logo/avatar upload fails with a permission error.
RUN mkdir -p /out/uploads/branding /out/uploads/avatars
FROM gcr.io/distroless/static-debian12
# distroless "nonroot" is uid/gid 65532; numeric --chown avoids needing the
# user to resolve during the copy.
COPY --from=build --chown=65532:65532 /out/uploads /uploads
COPY --from=build /out/api /api
ENV UPLOADS_DIR=/uploads
USER nonroot:nonroot
ENTRYPOINT ["/api"]
