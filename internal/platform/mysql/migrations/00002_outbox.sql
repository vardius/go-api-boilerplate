-- Messages waiting for delivery. The payload is kept as the exact JSON text
-- it was added with (the JSON type would reformat it), checked to be JSON.
-- No Down: rolling it back would lose undelivered messages.

-- +goose Up
CREATE TABLE outbox (
    id            BINARY(16)   NOT NULL,
    topic         VARCHAR(128) NOT NULL,
    payload       MEDIUMTEXT   CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    created_at    DATETIME(3)  NOT NULL,
    due_at        DATETIME(3)  NOT NULL,
    claimed_until DATETIME(3)  NULL,
    attempts      INT          NOT NULL DEFAULT 0,
    buried_at     DATETIME(3)  NULL,
    last_error    TEXT         NULL,
    PRIMARY KEY (id),
    KEY outbox_due (buried_at, due_at),
    CONSTRAINT outbox_payload_is_json CHECK (JSON_VALID(payload))
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;
