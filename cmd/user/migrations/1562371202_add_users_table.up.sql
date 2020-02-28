START TRANSACTION;
CREATE TABLE IF NOT EXISTS `users` (
    `distinctId`   INT                                    NOT NULL AUTO_INCREMENT,
    `id`           CHAR(36)                               NOT NULL,
    `name`         VARCHAR(255) COLLATE utf8_general_ci   NOT NULL, 
    `emailAddress` VARCHAR(255) COLLATE utf8_general_ci   NOT NULL,
    `password`     VARCHAR(255) COLLATE utf8_general_ci   NOT NULL,
    `facebookId`   VARCHAR(255)                                     DEFAULT NULL,
    `googleId`     VARCHAR(255)                                     DEFAULT NULL,
    PRIMARY KEY (`distinctId`),
    UNIQUE KEY `id` (`id`),
    UNIQUE KEY `emailAddress` (`emailAddress`),
    INDEX `i_facebookId` (`facebookId`),
    INDEX `i_googleId` (`googleId`)
)
ENGINE = InnoDB
DEFAULT CHARSET = utf8
COLLATE = utf8_bin;
COMMIT;
