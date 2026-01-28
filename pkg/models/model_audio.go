package models

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"time"
)

// Audio stores the metadata for a single audio file.
type Audio struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Date    *Date  `json:"date"`
	Details string `json:"details"`
	// Rating expressed in 1-100 scale
	Rating       *int    `json:"rating"`
	Organized    bool    `json:"organized"`
	ResumeTime   float64 `json:"resume_time"`
	PlayDuration float64 `json:"play_duration"`

	// transient - not persisted
	Files         RelatedFiles
	PrimaryFileID *FileID
	// transient - path of primary file - empty if no files
	Path string
	// transient - checksum of primary file - empty if no files
	Checksum string

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	URLs         RelatedStrings `json:"urls"`
	TagIDs       RelatedIDs     `json:"tag_ids"`
	PerformerIDs RelatedIDs     `json:"performer_ids"`
}

func NewAudio() Audio {
	currentTime := time.Now()
	return Audio{
		CreatedAt:    currentTime,
		UpdatedAt:    currentTime,
		Files:        NewRelatedFiles([]File{}),
		TagIDs:       NewRelatedIDs([]int{}),
		PerformerIDs: NewRelatedIDs([]int{}),
	}
}

// AudioPartial represents part of an Audio object. It is used to update
// the database entry.
type AudioPartial struct {
	Title   OptionalString
	Date    OptionalDate
	Details OptionalString
	// Rating expressed in 1-100 scale
	Rating        OptionalInt
	Organized     OptionalBool
	ResumeTime    OptionalFloat64
	PlayDuration  OptionalFloat64
	CreatedAt     OptionalTime
	UpdatedAt     OptionalTime
	URLs          *UpdateStrings
	TagIDs        *UpdateIDs
	PerformerIDs  *UpdateIDs
	PrimaryFileID *FileID
}

func NewAudioPartial() AudioPartial {
	currentTime := time.Now()
	return AudioPartial{
		UpdatedAt: NewOptionalTime(currentTime),
	}
}

func (a *Audio) LoadURLs(ctx context.Context, l URLLoader) error {
	return a.URLs.load(func() ([]string, error) {
		return l.GetURLs(ctx, a.ID)
	})
}

func (a *Audio) LoadFiles(ctx context.Context, l FileLoader) error {
	return a.Files.load(func() ([]File, error) {
		return l.GetFiles(ctx, a.ID)
	})
}

func (a *Audio) LoadPrimaryFile(ctx context.Context, l FileGetter) error {
	return a.Files.loadPrimary(func() (File, error) {
		if a.PrimaryFileID == nil {
			return nil, nil
		}

		f, err := l.Find(ctx, *a.PrimaryFileID)
		if err != nil {
			return nil, err
		}

		if len(f) > 0 {
			return f[0], nil
		}
		return nil, nil
	})
}

func (a *Audio) LoadTagIDs(ctx context.Context, l TagIDLoader) error {
	return a.TagIDs.load(func() ([]int, error) {
		return l.GetTagIDs(ctx, a.ID)
	})
}

func (a *Audio) LoadPerformerIDs(ctx context.Context, l PerformerIDLoader) error {
	return a.PerformerIDs.load(func() ([]int, error) {
		return l.GetPerformerIDs(ctx, a.ID)
	})
}

func (a *Audio) LoadRelationships(ctx context.Context, l AudioReader) error {
	if err := a.LoadURLs(ctx, l); err != nil {
		return err
	}

	if err := a.LoadTagIDs(ctx, l); err != nil {
		return err
	}

	if err := a.LoadPerformerIDs(ctx, l); err != nil {
		return err
	}

	if err := a.LoadFiles(ctx, l); err != nil {
		return err
	}

	return nil
}

// UpdateInput constructs an AudioUpdateInput using the populated fields in the AudioPartial object.
func (a AudioPartial) UpdateInput(id int) AudioUpdateInput {
	var dateStr *string
	if a.Date.Set {
		d := a.Date.Value
		v := d.String()
		dateStr = &v
	}

	ret := AudioUpdateInput{
		ID:           strconv.Itoa(id),
		Title:        a.Title.Ptr(),
		URLs:         a.URLs.Strings(),
		Date:         dateStr,
		Details:      a.Details.Ptr(),
		Rating100:    a.Rating.Ptr(),
		Organized:    a.Organized.Ptr(),
		ResumeTime:   a.ResumeTime.Ptr(),
		PlayDuration: a.PlayDuration.Ptr(),
	}

	if a.TagIDs != nil {
		ret.TagIds = a.TagIDs.IDStrings()
	}
	if a.PerformerIDs != nil {
		ret.PerformerIds = a.PerformerIDs.IDStrings()
	}

	return ret
}

// GetName returns the name of the audio or the filename if name is empty.
func (a Audio) GetName() string {
	if a.Title != "" {
		return a.Title
	}

	if a.Path != "" {
		return filepath.Base(a.Path)
	}

	return ""
}

// DisplayName returns a display name for the audio.
func (a Audio) DisplayName() string {
	if a.Path != "" {
		return a.Path
	}

	return strconv.Itoa(a.ID)
}

// Validate validates the audio data.
func (a Audio) Validate() error {
	if a.Title == "" && a.Path == "" {
		return errors.New("audio must have either title or path")
	}

	if a.Rating != nil && (*a.Rating < 1 || *a.Rating > 100) {
		return errors.New("rating must be between 1 and 100")
	}

	return nil
}

// AudioFileType represents metadata for an audio file in scraped/external data.
type AudioFileType struct {
	Size       *string  `graphql:"size" json:"size"`
	Duration   *float64 `graphql:"duration" json:"duration"`
	AudioCodec *string  `graphql:"audio_codec" json:"audio_codec"`
	Format     *string  `graphql:"format" json:"format"`
	Bitrate    *int     `graphql:"bitrate" json:"bitrate"`
	SampleRate *int     `graphql:"sample_rate" json:"sample_rate"`
	Channels   *int     `graphql:"channels" json:"channels"`
}
