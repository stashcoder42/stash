CREATE TABLE IF NOT EXISTS `audios` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255) not null,
  `date` date,
  `date_precision` tinyint,
  `rating` tinyint,
  `details` text,
  `organized` boolean not null default '0',
  `resume_time` float not null default 0,
  `play_duration` float not null default 0,
  `cover_blob` varchar(255) REFERENCES `blobs`(`checksum`),
  `created_at` datetime not null,
  `updated_at` datetime not null
);

-- Audio file metadata table (similar to video_files, image_files)
CREATE TABLE IF NOT EXISTS `audio_files` (
  `file_id` integer not null primary key,
  `format` varchar(255),
  `duration` float not null default 0,
  `audio_codec` varchar(255),
  `bitrate` integer not null default 0,
  `sample_rate` integer not null default 0,
  `channels` integer not null default 0,
  foreign key(`file_id`) references `files`(`id`) on delete CASCADE
);

-- Audio-to-file relationship table (similar to scenes_files, images_files)
CREATE TABLE IF NOT EXISTS `audios_files` (
  `audio_id` integer not null,
  `file_id` integer not null,
  `primary` boolean not null,
  foreign key(`audio_id`) references `audios`(`id`) on delete CASCADE,
  foreign key(`file_id`) references `files`(`id`) on delete CASCADE,
  PRIMARY KEY(`audio_id`, `file_id`)
);

CREATE TABLE IF NOT EXISTS `audio_performers` (
  `audio_id` integer not null,
  `performer_id` integer not null,
  foreign key(`audio_id`) references `audios`(`id`) on delete CASCADE,
  foreign key(`performer_id`) references `performers`(`id`) on delete CASCADE,
  PRIMARY KEY(`audio_id`, `performer_id`)
);

CREATE TABLE IF NOT EXISTS `audio_tags` (
  `audio_id` integer not null,
  `tag_id` integer not null,
  foreign key(`audio_id`) references `audios`(`id`) on delete CASCADE,
  foreign key(`tag_id`) references `tags`(`id`) on delete CASCADE,
  PRIMARY KEY(`audio_id`, `tag_id`)
);

-- Audio history tables (similar to scenes_view_dates and scenes_o_dates)
CREATE TABLE IF NOT EXISTS `audios_view_dates` (
  `audio_id` integer,
  `view_date` datetime not null,
  foreign key(`audio_id`) references `audios`(`id`) on delete CASCADE
);

CREATE TABLE IF NOT EXISTS `audios_o_dates` (
  `audio_id` integer,
  `o_date` datetime not null,
  foreign key(`audio_id`) references `audios`(`id`) on delete CASCADE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS `index_audios_files_audio_id` ON `audios_files` (`audio_id`);
CREATE INDEX IF NOT EXISTS `index_audios_files_file_id` ON `audios_files` (`file_id`);
CREATE UNIQUE INDEX IF NOT EXISTS `unique_index_audios_files_on_primary` ON `audios_files` (`audio_id`) WHERE `primary` = 1;
CREATE INDEX IF NOT EXISTS `index_audio_performers_audio_id` ON `audio_performers` (`audio_id`);
CREATE INDEX IF NOT EXISTS `index_audio_performers_performer_id` ON `audio_performers` (`performer_id`);
CREATE INDEX IF NOT EXISTS `index_audio_tags_audio_id` ON `audio_tags` (`audio_id`);
CREATE INDEX IF NOT EXISTS `index_audio_tags_tag_id` ON `audio_tags` (`tag_id`);
CREATE INDEX IF NOT EXISTS `index_audios_title` ON `audios` (`title`);
CREATE INDEX IF NOT EXISTS `index_audios_date` ON `audios` (`date`);
CREATE INDEX IF NOT EXISTS `index_audios_rating` ON `audios` (`rating`);
CREATE INDEX IF NOT EXISTS `index_audios_view_dates` ON `audios_view_dates` (`audio_id`);
CREATE INDEX IF NOT EXISTS `index_audios_o_dates` ON `audios_o_dates` (`audio_id`);

CREATE TABLE IF NOT EXISTS `audio_urls` (
  `audio_id` integer NOT NULL,
  `position` integer NOT NULL,
  `url` varchar(255) NOT NULL,
  foreign key(`audio_id`) references `audios`(`id`) on delete CASCADE,
  PRIMARY KEY(`audio_id`, `position`, `url`)
);

CREATE INDEX IF NOT EXISTS `audio_urls_url` on `audio_urls` (`url`);
