START TRANSACTION;
CREATE TABLE IF NOT EXISTS `events`
(
    `distinct_id`    INT          NOT NULL AUTO_INCREMENT,
    `event_id`       CHAR(36)     NOT NULL,
    `event_type`     VARCHAR(255) NOT NULL,
    `stream_id`      CHAR(36)     NOT NULL,
    `stream_name`    VARCHAR(255) NOT NULL,
    `stream_version` INT          NOT NULL,
    `occurred_at`    DATETIME     NOT NULL,
    `payload`        JSON         NOT NULL,
    `metadata`       JSON DEFAULT NULL,
    PRIMARY KEY (`distinct_id`),
    UNIQUE KEY `u_event_id` (`event_id`),
    INDEX `i_stream_id_stream_name_event_type` (`stream_id`, `stream_name`, `event_type`)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
COMMIT;
