FROM migrate/migrate:v4.7.1

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ENV BIN=${BIN}

RUN mkdir -p /migrations

COPY cmd/auth/migrations /migrations
COPY cmd/user/migrations /migrations

ENTRYPOINT ["/migrate"]
CMD ["--help"]
