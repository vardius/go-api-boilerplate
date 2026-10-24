-- The event log. Events are never changed or deleted, so this migration has
-- no Down: rolling it back would destroy history.

-- +goose Up
CREATE TABLE events (
    position    BIGINT       NOT NULL AUTO_INCREMENT,
    id          BINARY(16)   NOT NULL,
    stream_id   BINARY(16)   NOT NULL,
    stream_type VARCHAR(64)  NOT NULL,
    version     INT          NOT NULL,
    type        VARCHAR(128) NOT NULL,
    payload     JSON         NOT NULL,
    meta        JSON         NOT NULL,
    occurred_at DATETIME(3)  NOT NULL,
    PRIMARY KEY (position),
    UNIQUE KEY events_id (id),
    UNIQUE KEY events_stream_version (stream_id, version)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;
