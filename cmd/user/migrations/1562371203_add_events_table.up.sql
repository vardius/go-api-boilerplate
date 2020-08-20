START TRANSACTION;
CREATE TABLE IF NOT EXISTS `events`
(
    `distinctId`    INT          NOT NULL AUTO_INCREMENT,
    `id`            CHAR(36)     NOT NULL,
    `type`          VARCHAR(255) NOT NULL,
    `streamId`      CHAR(36)     NOT NULL,
    `streamName`    VARCHAR(255) NOT NULL,
    `streamVersion` INT          NOT NULL,
    `occurredAt`    DATETIME DEFAULT NULL,
    `payload`       JSON     DEFAULT NULL,
    PRIMARY KEY (`distinctId`),
    UNIQUE KEY `u_id` (`id`),
    INDEX `i_stream_id_stream_name_type` (`streamId`, `streamName`, `type`)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
COMMIT;
