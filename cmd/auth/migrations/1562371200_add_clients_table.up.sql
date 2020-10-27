START TRANSACTION;
CREATE TABLE IF NOT EXISTS `clients`
(
    `distinct_id`  INT          NOT NULL AUTO_INCREMENT,
    `id`           CHAR(36)     NOT NULL,
    `user_id`      CHAR(36)     NOT NULL,
    `secret`       VARCHAR(255) NOT NULL,
    `domain`       VARCHAR(255) NOT NULL,
    `redirect_url` TEXT         NOT NULL,
    `scope`        JSON         NOT NULL,
    PRIMARY KEY (`distinct_id`),
    UNIQUE KEY `id` (`id`),
    INDEX `i_user_id_domain` (`user_id`, `domain`)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
COMMIT;
