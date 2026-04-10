package models

import "context"

type AudioFilterType struct {
	OperatorFilter[AudioFilterType]
	ID      *IntCriterionInput    `json:"id"`
	Title   *StringCriterionInput `json:"title"`
	Details *StringCriterionInput `json:"details"`
	// Filter by file checksum
	Checksum *StringCriterionInput `json:"checksum"`
	// Filter by path
	Path *StringCriterionInput `json:"path"`
	// Filter by file count
	FileCount *IntCriterionInput `json:"file_count"`
	// Filter by rating expressed as 1-100
	Rating100 *IntCriterionInput `json:"rating100"`
	// Filter by organized
	Organized *bool `json:"organized"`
	// Filter by o_counter
	OCounter *IntCriterionInput `json:"o_counter"`
	// Filter by duration (in seconds)
	Duration *IntCriterionInput `json:"duration"`
	// Filter by bitrate
	Bitrate *IntCriterionInput `json:"bitrate"`
	// Filter by audio codec
	AudioCodec *StringCriterionInput `json:"audio_codec"`
	// Filter by sample rate
	SampleRate *IntCriterionInput `json:"sample_rate"`
	// Filter by number of channels
	Channels *IntCriterionInput `json:"channels"`
	// Filter by resume time
	ResumeTime *FloatCriterionInput `json:"resume_time"`
	// Filter by play count
	PlayCount *IntCriterionInput `json:"play_count"`
	// Filter by play duration (in seconds)
	PlayDuration *FloatCriterionInput `json:"play_duration"`
	// Filter by last played at
	LastPlayedAt *TimestampCriterionInput `json:"last_played_at"`
	// Filter to only include audio missing this property
	IsMissing *string `json:"is_missing"`
	// Filter to only include audio with these tags
	Tags *HierarchicalMultiCriterionInput `json:"tags"`
	// Filter by tag count
	TagCount *IntCriterionInput `json:"tag_count"`
	// Filter to only include audio with performers with these tags
	PerformerTags *HierarchicalMultiCriterionInput `json:"performer_tags"`
	// Filter audio that have performers that have been favorited
	PerformerFavorite *bool `json:"performer_favorite"`
	// Filter to only include audio with these performers
	Performers *MultiCriterionInput `json:"performers"`
	// Filter by performer count
	PerformerCount *IntCriterionInput `json:"performer_count"`
	// Filter audios by performer age at time of audio
	PerformerAge *IntCriterionInput `json:"performer_age"`
	// Filter by url
	URL *StringCriterionInput `json:"url"`
	// Filter by date
	Date *DateCriterionInput `json:"date"`
	// Filter by related performers that meet this criteria
	PerformersFilter *PerformerFilterType `json:"performers_filter"`
	// Filter by related tags that meet this criteria
	TagsFilter *TagFilterType `json:"tags_filter"`
	// Filter by created at
	CreatedAt *TimestampCriterionInput `json:"created_at"`
	// Filter by updated at
	UpdatedAt *TimestampCriterionInput `json:"updated_at"`
	// Filter by custom fields
	CustomFields []CustomFieldCriterionInput `json:"custom_fields"`
}

type AudioQueryOptions struct {
	QueryOptions
	AudioFilter *AudioFilterType

	TotalDuration bool
	TotalSize     bool
}

type AudioQueryResult struct {
	QueryResult[int]
	TotalDuration float64
	TotalSize     float64

	getter     AudioGetter
	audios     []*Audio
	resolveErr error
}

type AudioCreateInput struct {
	Title        *string  `json:"title"`
	Date         *string  `json:"date"`
	Details      *string  `json:"details"`
	Rating100    *int     `json:"rating100"`
	Organized    *bool    `json:"organized"`
	ResumeTime   *float64 `json:"resume_time"`
	PlayDuration *float64 `json:"play_duration"`
	CoverImage   *string  `json:"cover_image"`

	PerformerIds []string `json:"performer_ids"`
	TagIds       []string `json:"tag_ids"`

	// The first id will be assigned as primary.
	// Files will be reassigned from existing audio if applicable.
	// Files must not already be primary for another audio.
	FileIds      []string               `json:"file_ids"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

type AudioUpdateInput struct {
	ClientMutationID *string  `json:"clientMutationId"`
	ID               string   `json:"id"`
	Title            *string  `json:"title"`
	URLs             []string `json:"urls"`
	Date             *string  `json:"date"`
	Details          *string  `json:"details"`
	Rating100        *int     `json:"rating100"`
	Organized        *bool    `json:"organized"`
	ResumeTime       *float64 `json:"resume_time"`
	PlayDuration     *float64 `json:"play_duration"`
	PlayCount        *int     `json:"play_count"`
	CoverImage       *string  `json:"cover_image"`

	PerformerIds  []string           `json:"performer_ids"`
	TagIds        []string           `json:"tag_ids"`
	PrimaryFileID *string            `json:"primary_file_id"`
	CustomFields  *CustomFieldsInput `json:"custom_fields"`
}

type AudioDestroyInput struct {
	ID              string `json:"id"`
	DeleteFile      *bool  `json:"delete_file"`
	DeleteGenerated *bool  `json:"delete_generated"`
}

type AudiosDestroyInput struct {
	Ids             []string `json:"ids"`
	DeleteFile      *bool    `json:"delete_file"`
	DeleteGenerated *bool    `json:"delete_generated"`
}

func NewAudioQueryResult(getter AudioGetter) *AudioQueryResult {
	return &AudioQueryResult{
		getter: getter,
	}
}

func (r *AudioQueryResult) Resolve(ctx context.Context) ([]*Audio, error) {
	// cache results
	if r.audios == nil && r.resolveErr == nil {
		r.audios, r.resolveErr = r.getter.FindMany(ctx, r.IDs)
	}
	return r.audios, r.resolveErr
}
