import {
  createMandatoryNumberCriterionOption,
  createStringCriterionOption,
  createDateCriterionOption,
  createMandatoryTimestampCriterionOption,
  createDurationCriterionOption,
} from "./criteria/criterion";
import { HasMarkersCriterionOption } from "./criteria/has-markers";
import { OrganizedCriterionOption } from "./criteria/organized";
import { PerformersCriterionOption } from "./criteria/performers";
import {
  PerformerTagsCriterionOption,
  TagsCriterionOption,
} from "./criteria/tags";
import { ListFilterOptions, MediaSortByOptions } from "./filter-options";
import { DisplayMode } from "./types";
import { PerformerFavoriteCriterionOption } from "./criteria/favorite";
import { StashIDCriterionOption } from "./criteria/stash-ids";
import { RatingCriterionOption } from "./criteria/rating";
import { PathCriterionOption } from "./criteria/path";
import { CustomFieldsCriterionOption } from "./criteria/custom-fields";

const defaultSortBy = "title";
const sortByOptions = [
  "organized",
  "date",
  "file_count",
  "filesize",
  "duration",
  "bitrate",
  "sample_rate",
  "channels",
  ...MediaSortByOptions,
]
  .map(ListFilterOptions.createSortBy)
  .concat([
    {
      messageID: "o_count",
      value: "o_counter",
    },
    {
      messageID: "last_o_at",
      value: "last_o_at",
      sfwMessageID: "last_o_at_sfw",
    },
  ]);

// Audio supports neither the Wall nor the Tagger display mode - both are
// non-goals, see docs/AUDIO_SCENE_PARITY.md "Audio non-goals".
//
// Wall exists to play many animated previews at once, muted, with hover to
// unmute one tile. Audio has no animated asset, and a listener cannot pick
// one stream out of thirty. Tagger is blocked on stash-box, which has no
// audio entity to match against.
const displayModeOptions = [DisplayMode.Grid, DisplayMode.List];

export const DurationCriterionOption =
  createDurationCriterionOption("duration");

const criterionOptions = [
  createStringCriterionOption("title"),
  PathCriterionOption,
  createStringCriterionOption("details"),
  createStringCriterionOption("checksum", "media_info.checksum"),
  OrganizedCriterionOption,
  RatingCriterionOption,
  createMandatoryNumberCriterionOption("o_counter", "o_count"),
  DurationCriterionOption,
  createMandatoryNumberCriterionOption("bitrate"),
  createMandatoryNumberCriterionOption("sample_rate"),
  createMandatoryNumberCriterionOption("channels"),
  createStringCriterionOption("audio_codec"),
  HasMarkersCriterionOption,
  TagsCriterionOption,
  createMandatoryNumberCriterionOption("tag_count"),
  PerformerTagsCriterionOption,
  PerformersCriterionOption,
  createMandatoryNumberCriterionOption("performer_count"),
  createMandatoryNumberCriterionOption("performer_age"),
  PerformerFavoriteCriterionOption,
  createStringCriterionOption("url"),
  StashIDCriterionOption,
  createMandatoryNumberCriterionOption("file_count"),
  createDateCriterionOption("date"),
  createMandatoryTimestampCriterionOption("created_at"),
  createMandatoryTimestampCriterionOption("updated_at"),
  CustomFieldsCriterionOption,
];

export const AudioListFilterOptions = new ListFilterOptions(
  defaultSortBy,
  sortByOptions,
  displayModeOptions,
  criterionOptions
);
