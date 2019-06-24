START TRANSACTION;
CREATE TABLE `auth_tokens` (
    `distinctId` INT          NOT NULL AUTO_INCREMENT,
    `id`         CHAR(36)     NOT NULL,
    `clientId`   CHAR(36)     NOT NULL,
    `userId`     CHAR(36)     NOT NULL,
    `code`       VARCHAR(255) DEFAULT NULL,
    `access`     TEXT         NOT NULL,
    `refresh`    TEXT         NOT NULL,
    `data`       JSON         NOT NULL,
    PRIMARY KEY (`distinctId`),
    UNIQUE KEY `id` (`id`),
    INDEX `i_userId` (`userId`),
    INDEX `i_code` (`code`),
    INDEX `i_access` (`access`),
    INDEX `i_refresh` (`refresh`)
)
ENGINE = InnoDB
DEFAULT CHARSET = utf8
COLLATE = utf8_bin;
COMMIT;
