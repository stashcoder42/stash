CREATE TABLE IF NOT EXISTS `audio_custom_fields` (
  `audio_id` integer NOT NULL,
  `field` varchar(64) NOT NULL,
  `value` BLOB NOT NULL,
  PRIMARY KEY (`audio_id`, `field`),
  foreign key(`audio_id`) references `audios`(`id`) on delete CASCADE
);
CREATE INDEX IF NOT EXISTS `index_audio_custom_fields_field_value`
  ON `audio_custom_fields` (`field`, `value`);
