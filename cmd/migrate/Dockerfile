FROM migrate/migrate:v4.7.1

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ENV BIN=${BIN}

# Copy migration files into the container.
COPY cmd/migrate/migrations migrations

ENTRYPOINT ["/migrate"]
CMD ["--help"]
