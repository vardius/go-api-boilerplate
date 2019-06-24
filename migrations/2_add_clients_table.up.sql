START TRANSACTION;
CREATE TABLE `clients` (
    `distinctId` INT          NOT NULL AUTO_INCREMENT,
    `id`         CHAR(36)     NOT NULL,
    `userId`     CHAR(36)     NOT NULL,
    `secret`     VARCHAR(255) NOT NULL,
    `domain`     VARCHAR(255) NOT NULL,
    `data`       JSON         NOT NULL,
    PRIMARY KEY (`distinctId`),
    UNIQUE KEY `id` (`id`),
    INDEX `i_userId` (`userId`)
)
ENGINE = InnoDB
DEFAULT CHARSET = utf8
COLLATE = utf8_bin;
COMMIT;
