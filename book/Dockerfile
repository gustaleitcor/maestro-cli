# syntax=docker/dockerfile:1
#
# Builds the Maestro book with book/build.sh, as a release does, into /out,
# then exits: the reverse proxy serves it at /docs/ (bind-mount book/dist on
# the host). Its context is maestro-cli. maestro carries the same book, for
# maestro docs to open offline; this copy is for maestro docs --web and the
# Maestro page's Docs.

# book/build.sh downloads mdbook-mermaid, which has no ARM build; the book is
# files, so amd64 serves anywhere.
FROM --platform=linux/amd64 docker.io/library/golang:1.27-bookworm AS book
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN sh book/build.sh

FROM docker.io/library/debian:bookworm-slim
COPY --from=book /src/book/dist /book
# Replaces what /out held with the book.
CMD ["sh", "-c", "find /out -mindepth 1 -delete && cp -a /book/. /out/"]
