START TRANSACTION;
CREATE TABLE IF NOT EXISTS `auth_tokens`
(
    `distinct_id` INT      NOT NULL AUTO_INCREMENT,
    `id`          CHAR(36) NOT NULL,
    `client_id`   CHAR(36) NOT NULL,
    `user_id`     CHAR(36) NOT NULL,
    `code`        VARCHAR(255) DEFAULT NULL,
    `access`      TEXT     NOT NULL,
    `refresh`     TEXT         DEFAULT NULL,
    `expired_at`  DATETIME NOT NULL,
    `user_agent`  TEXT         DEFAULT NULL,
    `data`        JSON     NOT NULL,
    PRIMARY KEY (`distinct_id`),
    INDEX `i_userId` (`user_id`),
    INDEX `i_code` (`code`)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
COMMIT;
