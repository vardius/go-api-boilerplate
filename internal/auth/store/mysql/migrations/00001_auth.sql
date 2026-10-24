-- The auth service's tables. Tokens and codes are kept only as SHA-256
-- hashes, the primary keys their lookups use. IDs that clients send are
-- compared byte for byte (utf8mb4_bin), as the other stores compare them.
-- No Down: rolling it back would delete every client and sign everyone out.

-- +goose Up
CREATE TABLE auth_clients (
    id            VARCHAR(64)  CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    owner_id      BINARY(16)   NOT NULL,
    name          VARCHAR(200) NOT NULL,
    redirect_uris JSON         NOT NULL,
    scopes        JSON         NOT NULL,
    secret_hash   VARCHAR(255) NOT NULL,
    created_at    DATETIME(3)  NOT NULL,
    PRIMARY KEY (id),
    KEY auth_clients_owner (owner_id, created_at, id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE auth_grants (
    id           BINARY(16)   NOT NULL,
    user_id      BINARY(16)   NOT NULL,
    client_id    VARCHAR(64)  CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    scopes       JSON         NOT NULL,
    user_agent   VARCHAR(512) NOT NULL,
    created_at   DATETIME(3)  NOT NULL,
    last_used_at DATETIME(3)  NOT NULL,
    expires_at   DATETIME(3)  NOT NULL,
    PRIMARY KEY (id),
    KEY auth_grants_user (user_id, client_id, last_used_at, id),
    KEY auth_grants_client (client_id),
    KEY auth_grants_expires (expires_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- A grant's tokens go with it: revoking or purging a grant is deleting it.
CREATE TABLE auth_refresh_tokens (
    hash       BINARY(32)  NOT NULL,
    grant_id   BINARY(16)  NOT NULL,
    expires_at DATETIME(3) NOT NULL,
    rotated    BOOLEAN     NOT NULL DEFAULT FALSE,
    PRIMARY KEY (hash),
    KEY auth_refresh_tokens_grant (grant_id),
    KEY auth_refresh_tokens_expires (expires_at),
    CONSTRAINT auth_refresh_tokens_grant FOREIGN KEY (grant_id) REFERENCES auth_grants (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE auth_authorization_requests (
    id             BINARY(16)  NOT NULL,
    user_id        BINARY(16)  NOT NULL,
    client_id      VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    redirect_uri   TEXT        NOT NULL,
    scopes         JSON        NOT NULL,
    state          TEXT        NOT NULL,
    code_challenge VARCHAR(64) NOT NULL,
    expires_at     DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    KEY auth_authorization_requests_client (client_id),
    KEY auth_authorization_requests_expires (expires_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE auth_authorization_codes (
    hash           BINARY(32)  NOT NULL,
    client_id      VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    user_id        BINARY(16)  NOT NULL,
    redirect_uri   TEXT        NOT NULL,
    scopes         JSON        NOT NULL,
    code_challenge VARCHAR(64) NOT NULL,
    expires_at     DATETIME(3) NOT NULL,
    used           BOOLEAN     NOT NULL DEFAULT FALSE,
    grant_id       BINARY(16)  NULL,
    PRIMARY KEY (hash),
    KEY auth_authorization_codes_client (client_id),
    KEY auth_authorization_codes_expires (expires_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;
