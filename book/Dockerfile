# syntax=docker/dockerfile:1
#
# Copies the Maestro book, as book/build.sh built it into internal/book/site,
# into /out, then exits: the reverse proxy serves it at /docs/ (bind-mount
# the host's copy on /out). Its context is maestro-cli. maestro carries the
# same pages, for maestro docs to open offline; this copy is for maestro docs
# --web and the Maestro page's Docs.
FROM docker.io/library/debian:bookworm-slim
COPY internal/book/site /book
# Replaces what /out held with the book.
CMD ["sh", "-c", "find /out -mindepth 1 -delete && cp -a /book/. /out/"]
