CREATE TABLE IF NOT EXISTS `audio_markers` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255) not null,
  `seconds` float not null,
  `end_seconds` float,
  `primary_tag_id` integer not null,
  `audio_id` integer,
  `created_at` datetime not null,
  `updated_at` datetime not null,
  foreign key(`primary_tag_id`) references `tags`(`id`),
  foreign key(`audio_id`) references `audios`(`id`) on delete CASCADE
);

CREATE TABLE IF NOT EXISTS `audio_markers_tags` (
  `audio_marker_id` integer,
  `tag_id` integer,
  foreign key(`audio_marker_id`) references `audio_markers`(`id`) on delete CASCADE,
  foreign key(`tag_id`) references `tags`(`id`)
);

CREATE INDEX IF NOT EXISTS `index_audio_markers_tags_on_tag_id` on `audio_markers_tags` (`tag_id`);
CREATE INDEX IF NOT EXISTS `index_audio_markers_tags_on_audio_marker_id` on `audio_markers_tags` (`audio_marker_id`);
CREATE INDEX IF NOT EXISTS `index_audio_markers_on_audio_id` on `audio_markers` (`audio_id`);
CREATE INDEX IF NOT EXISTS `index_audio_markers_on_primary_tag_id` on `audio_markers` (`primary_tag_id`);
