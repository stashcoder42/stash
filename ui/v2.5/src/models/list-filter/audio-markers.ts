import { PerformersCriterionOption } from "./criteria/performers";
import { TagsCriterionOption } from "./criteria/tags";
import { ListFilterOptions } from "./filter-options";
import { DisplayMode } from "./types";
import {
  createMandatoryTimestampCriterionOption,
} from "./criteria/criterion";

const defaultSortBy = "title";
const sortByOptions = [
  "title",
  "seconds",
  "random",
  "created_at",
  "updated_at",
].map(ListFilterOptions.createSortBy);
const displayModeOptions = [DisplayMode.Grid];
const criterionOptions = [
  TagsCriterionOption,
  PerformersCriterionOption,
  createMandatoryTimestampCriterionOption("created_at"),
  createMandatoryTimestampCriterionOption("updated_at"),
];

export const AudioMarkerListFilterOptions = new ListFilterOptions(
  defaultSortBy,
  sortByOptions,
  displayModeOptions,
  criterionOptions
);
