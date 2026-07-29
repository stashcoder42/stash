package audio

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/json"
	"github.com/stashapp/stash/pkg/models/jsonschema"
	"github.com/stashapp/stash/pkg/sliceutil"
)

type TagFinder interface {
	models.TagGetter
	FindByAudioID(ctx context.Context, audioID int) ([]*models.Tag, error)
}

type PerformerFinder interface {
	models.PerformerGetter
	FindByAudioID(ctx context.Context, audioID int) ([]*models.Performer, error)
}

// ToBasicJSON converts an audio object into its JSON object equivalent. It
// does not convert the relationships to other objects.
func ToBasicJSON(audio *models.Audio) *jsonschema.Audio {
	newAudioJSON := jsonschema.Audio{
		Title:     audio.Title,
		Details:   audio.Details,
		Organized: audio.Organized,
		CreatedAt: json.JSONTime{Time: audio.CreatedAt},
		UpdatedAt: json.JSONTime{Time: audio.UpdatedAt},
	}

	// Use the first URL from the URLs list for backwards compatibility
	if audio.URLs.Loaded() && len(audio.URLs.List()) > 0 {
		newAudioJSON.URL = audio.URLs.List()[0]
	}

	if audio.Rating != nil {
		newAudioJSON.Rating = *audio.Rating
	}

	if audio.Date != nil {
		newAudioJSON.Date = audio.Date.String()
	}

	for _, f := range audio.Files.List() {
		newAudioJSON.Files = append(newAudioJSON.Files, f.Base().Path)
	}

	return &newAudioJSON
}

// GetTagNames returns a slice of tag names corresponding to the provided
// audio's tags.
func GetTagNames(ctx context.Context, reader TagFinder, audio *models.Audio) ([]string, error) {
	tags, err := reader.FindByAudioID(ctx, audio.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting audio tags: %w", err)
	}

	return getTagNames(tags), nil
}

func getTagNames(tags []*models.Tag) []string {
	var results []string
	for _, tag := range tags {
		if tag.Name != "" {
			results = append(results, tag.Name)
		}
	}

	return results
}

// GetPerformerNames returns a slice of performer names corresponding to the provided
// audio's performers.
func GetPerformerNames(ctx context.Context, reader PerformerFinder, audio *models.Audio) ([]string, error) {
	performers, err := reader.FindByAudioID(ctx, audio.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting audio performers: %w", err)
	}

	return getPerformerNames(performers), nil
}

func getPerformerNames(performers []*models.Performer) []string {
	var results []string
	for _, performer := range performers {
		if performer.Name != "" {
			results = append(results, performer.Name)
		}
	}

	return results
}

// GetDependentTagIDs returns a slice of unique tag IDs that this audio references.
func GetDependentTagIDs(ctx context.Context, tags TagFinder, audio *models.Audio) ([]int, error) {
	var ret []int

	t, err := tags.FindByAudioID(ctx, audio.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting audio tags: %w", err)
	}

	for _, tt := range t {
		ret = sliceutil.AppendUnique(ret, tt.ID)
	}

	return ret, nil
}

// GetDependentPerformerIDs returns a slice of unique performer IDs that this audio references.
func GetDependentPerformerIDs(ctx context.Context, performers PerformerFinder, audio *models.Audio) ([]int, error) {
	var ret []int

	p, err := performers.FindByAudioID(ctx, audio.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting audio performers: %w", err)
	}

	for _, pp := range p {
		ret = sliceutil.AppendUnique(ret, pp.ID)
	}

	return ret, nil
}
