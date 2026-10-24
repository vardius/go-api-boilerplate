-- One-time login codes, by the SHA-256 of the code: the code itself is
-- never stored. used_at is set once, when the code is redeemed.

-- +goose Up
CREATE TABLE login_codes (
    code_hash  BINARY(32)    NOT NULL,
    purpose    VARCHAR(32)   CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    email      VARCHAR(254)  CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    user_id    BINARY(16)    NULL,
    `next`     VARCHAR(2048) NOT NULL DEFAULT '',
    expires_at DATETIME(3)   NOT NULL,
    used_at    DATETIME(3)   NULL,
    PRIMARY KEY (code_hash),
    KEY login_codes_expires_at (expires_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE login_codes;
