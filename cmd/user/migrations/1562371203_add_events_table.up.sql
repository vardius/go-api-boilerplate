START TRANSACTION;
CREATE TABLE IF NOT EXISTS `events`
(
    `distinctId`    INT          NOT NULL AUTO_INCREMENT,
    `event_id`      CHAR(36)     NOT NULL,
    `event_type`    VARCHAR(255) NOT NULL,
    `streamId`      CHAR(36)     NOT NULL,
    `streamName`    VARCHAR(255) NOT NULL,
    `streamVersion` INT          NOT NULL,
    `occurredAt`    DATETIME DEFAULT NULL,
    `payload`       JSON     DEFAULT NULL,
    PRIMARY KEY (`distinctId`),
    UNIQUE KEY `u_event_id` (`event_id`),
    INDEX `i_stream_id_stream_name_event_type_event_id` (`streamId`, `streamName`, `event_type`, `event_id`)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
COMMIT;
