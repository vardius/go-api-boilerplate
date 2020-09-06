START TRANSACTION;
CREATE TABLE IF NOT EXISTS `users`
(
    `distinct_id`   INT                                  NOT NULL AUTO_INCREMENT,
    `id`            CHAR(36)                             NOT NULL,
    `email_address` VARCHAR(255) COLLATE utf8_general_ci NOT NULL,
    `facebook_id`   VARCHAR(255) DEFAULT NULL,
    `google_id`     VARCHAR(255) DEFAULT NULL,
    PRIMARY KEY (`distinct_id`),
    UNIQUE KEY `id` (`id`),
    UNIQUE KEY `email_address` (`email_address`),
    INDEX `i_facebook_id` (`facebook_id`),
    INDEX `i_google_id` (`google_id`)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
COMMIT;
