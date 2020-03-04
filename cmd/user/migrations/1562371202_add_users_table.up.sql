START TRANSACTION;
CREATE TABLE IF NOT EXISTS `users` (
    `distinctId`   INT                                    NOT NULL AUTO_INCREMENT,
    `id`           CHAR(36)                               NOT NULL,
    `provider`     VARCHAR(255)                                     DEFAULT NULL,                                   
    `name`         VARCHAR(255) COLLATE utf8_general_ci   NOT NULL, 
    `emailAddress` VARCHAR(255) COLLATE utf8_general_ci   NOT NULL,
    `password`     VARCHAR(255) COLLATE utf8_general_ci             DEFAULT NULL,
    `nickName`     VARCHAR(255) COLLATE utf8_general_ci             DEFAULT NULL,
    `location`     VARCHAR(255) COLLATE utf8_general_ci             DEFAULT NULL,
    `avatarURL`    VARCHAR(255) COLLATE utf8_general_ci             DEFAULT NULL,
    `description`  VARCHAR(255) COLLATE utf8_general_ci             DEFAULT NULL,
    `userId`       VARCHAR(255) COLLATE utf8_general_ci             DEFAULT NULL,
    `refreshToken` VARCHAR(300) COLLATE utf8_general_ci             DEFAULT NULL,
    PRIMARY KEY (`distinctId`),
    UNIQUE KEY `id` (`id`),
    UNIQUE KEY `emailAddress` (`emailAddress`),
    INDEX `i_userId` (`userId`)
)
ENGINE = InnoDB
DEFAULT CHARSET = utf8
COLLATE = utf8_bin;
COMMIT;
