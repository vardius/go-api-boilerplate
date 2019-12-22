FROM migrate/migrate:v4.7.1

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN
ENV BIN=${BIN}

COPY migrations migrations

ENTRYPOINT ["/migrate"]
CMD ["--help"]
