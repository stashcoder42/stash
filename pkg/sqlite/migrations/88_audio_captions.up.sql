CREATE TABLE IF NOT EXISTS `audio_captions` (
  `file_id` integer NOT NULL,
  `language_code` varchar(255) NOT NULL,
  `filename` varchar(255) NOT NULL,
  `caption_type` varchar(255) NOT NULL,
  primary key (`file_id`, `language_code`, `caption_type`),
  foreign key(`file_id`) references `audio_files`(`file_id`) on delete CASCADE
);
