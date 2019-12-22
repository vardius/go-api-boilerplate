START TRANSACTION;
CREATE TABLE IF NOT EXISTS `auth_tokens` (
`distinctId` INT          NOT NULL AUTO_INCREMENT,
`id`         CHAR(36)     NOT NULL,
`clientId`   CHAR(36)     NOT NULL,
`userId`     CHAR(36)     NOT NULL,
`code`       VARCHAR(255) DEFAULT NULL,
`refresh`    VARCHAR(255) NOT NULL,
`access`     VARCHAR(300) NOT NULL,
`data`       JSON         NOT NULL,
PRIMARY KEY (`distinctId`),
UNIQUE KEY `id` (`id`),
INDEX `i_userId` (`userId`),
INDEX `i_code` (`code`),
INDEX `i_refresh` (`refresh`),
INDEX `i_access` (`access`)
)
ENGINE = InnoDB
DEFAULT CHARSET = utf8
COLLATE = utf8_bin;
COMMIT;
