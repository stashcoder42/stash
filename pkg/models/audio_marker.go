package models

type AudioMarkerFilterType struct {
	OperatorFilter[AudioMarkerFilterType]
	// Filter by audio marker ID
	ID *IntCriterionInput `json:"id"`
	// Filter by title
	Title *StringCriterionInput `json:"title"`
	// Filter by start/end time
	Seconds *FloatCriterionInput `json:"seconds"`
	// Filter to only include audio markers from these audios
	Audios *MultiCriterionInput `json:"audios"`
	// Filter to only include audio markers with these tags
	Tags *HierarchicalMultiCriterionInput `json:"tags"`
	// Filter to only include audio markers attached to an audio with these tags
	AudioTags *HierarchicalMultiCriterionInput `json:"audio_tags"`
	// Filter to only include audio markers with these performers
	Performers *MultiCriterionInput `json:"performers"`
	// Filter by tag count
	TagCount *IntCriterionInput `json:"tag_count"`
	// Filter by duration (in seconds)
	Duration *FloatCriterionInput `json:"duration"`
	// Filter by created at
	CreatedAt *TimestampCriterionInput `json:"created_at"`
	// Filter by updated at
	UpdatedAt *TimestampCriterionInput `json:"updated_at"`
	// Filter by audio date
	AudioDate *DateCriterionInput `json:"audio_date"`
	// Filter by audio creation time
	AudioCreatedAt *TimestampCriterionInput `json:"audio_created_at"`
	// Filter by audio last update time
	AudioUpdatedAt *TimestampCriterionInput `json:"audio_updated_at"`
	// Filter by related audios that meet this criteria
	AudioFilter *AudioFilterType `json:"audio_filter"`
	// Filter by related audios that meet this criteria (deprecated)
	AudiosFilter *AudioFilterType `json:"audios_filter"`
	// Filter by related tags that meet this criteria
	TagsFilter *TagFilterType `json:"tags_filter"`
}
