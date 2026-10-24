-- The users projection, rebuilt from the user streams. Addresses arrive
-- lower-cased, so they are compared byte for byte: a case- or
-- accent-insensitive collation would call two different addresses one.
-- MySQL lets a unique key hold many NULLs, so unlinked users never clash on
-- google_sub.

-- +goose Up
CREATE TABLE users (
    id         BINARY(16)   NOT NULL,
    email      VARCHAR(254) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    google_sub VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NULL,
    created_at DATETIME(3)  NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY users_email (email),
    UNIQUE KEY users_google_sub (google_sub)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE users;
