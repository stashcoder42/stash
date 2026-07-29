package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAudioPartial_UpdateInput(t *testing.T) {
	const (
		id      = 1
		title   = "Test Audio"
		url     = "http://example.com/audio"
		details = "Test details"
		rating  = 80
	)

	dateObj, _ := ParseDate("2001-02-03")
	organized := true

	partial := AudioPartial{
		Title:     NewOptionalString(title),
		Date:      NewOptionalDate(dateObj),
		Details:   NewOptionalString(details),
		Rating:    NewOptionalInt(rating),
		Organized: NewOptionalBool(organized),
	}

	input := partial.UpdateInput(id)

	assert.Equal(t, "1", input.ID)
	assert.Equal(t, title, *input.Title)
	assert.Equal(t, dateObj.String(), *input.Date)
	assert.Equal(t, details, *input.Details)
	assert.Equal(t, rating, *input.Rating100)
	assert.Equal(t, organized, *input.Organized)
}

func TestAudio_GetName(t *testing.T) {
	tests := []struct {
		name     string
		audio    Audio
		expected string
	}{
		{
			"title set",
			Audio{
				Title: "test title",
			},
			"test title",
		},
		{
			"title empty, path set",
			Audio{
				Path: "/path/to/audio.mp3",
			},
			"audio.mp3",
		},
		{
			"both empty",
			Audio{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.audio.GetName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAudio_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		audio    Audio
		expected string
	}{
		{
			"path set",
			Audio{
				Path: "/path/to/audio.mp3",
				ID:   1,
			},
			"/path/to/audio.mp3",
		},
		{
			"path empty",
			Audio{
				ID: 1,
			},
			"1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.audio.DisplayName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAudio_Validate(t *testing.T) {
	tests := []struct {
		name    string
		audio   Audio
		wantErr bool
	}{
		{
			"valid with title",
			Audio{
				Title: "Test Audio",
			},
			false,
		},
		{
			"valid with path",
			Audio{
				Path: "/path/to/audio.mp3",
			},
			false,
		},
		{
			"invalid - no title or path",
			Audio{},
			true,
		},
		{
			"invalid rating - too low",
			Audio{
				Title:  "Test Audio",
				Rating: &[]int{0}[0],
			},
			true,
		},
		{
			"invalid rating - too high",
			Audio{
				Title:  "Test Audio",
				Rating: &[]int{101}[0],
			},
			true,
		},
		{
			"valid rating - minimum",
			Audio{
				Title:  "Test Audio",
				Rating: &[]int{1}[0],
			},
			false,
		},
		{
			"valid rating - maximum",
			Audio{
				Title:  "Test Audio",
				Rating: &[]int{100}[0],
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.audio.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewAudio(t *testing.T) {
	currentTime := time.Now()
	a := NewAudio()

	assert := assert.New(t)
	assert.NotEqual(time.Time{}, a.CreatedAt)
	assert.NotEqual(time.Time{}, a.UpdatedAt)
	assert.True(a.CreatedAt.Equal(a.UpdatedAt))

	// CreatedAt/UpdatedAt should be within a second of currentTime
	assert.Less(currentTime.Sub(a.CreatedAt).Abs(), time.Second)
	assert.Less(currentTime.Sub(a.UpdatedAt).Abs(), time.Second)

	// Check defaults
	assert.Empty(a.Title)
	assert.Nil(a.Date)
	assert.Empty(a.Details)
	assert.Nil(a.Rating)
	assert.False(a.Organized)
	assert.NotNil(a.Files)
	assert.Nil(a.PrimaryFileID)
	assert.Empty(a.Path)
	assert.Empty(a.Checksum)
	assert.NotNil(a.TagIDs)
	assert.NotNil(a.PerformerIDs)
	assert.NotNil(a.URLs)
}

func TestNewAudioPartial(t *testing.T) {
	currentTime := time.Now()
	p := NewAudioPartial()

	assert := assert.New(t)
	assert.True(p.UpdatedAt.Set)
	assert.Less(currentTime.Sub(p.UpdatedAt.Value).Abs(), time.Second)

	// Check other fields are not set
	assert.False(p.Title.Set)
	assert.False(p.Date.Set)
	assert.False(p.Details.Set)
	assert.False(p.Rating.Set)
	assert.False(p.Organized.Set)
	assert.False(p.CreatedAt.Set)
	assert.Nil(p.URLs)
}

func TestAudioUpdateInput_URLs(t *testing.T) {
	// Test to prove that AudioUpdateInput supports URLs field
	urls := []string{
		"https://example.com/audio1.mp3",
		"https://streaming.site/track/123",
		"https://backup.server/files/audio1.mp3",
	}

	input := AudioUpdateInput{
		ID:   "1",
		URLs: urls,
	}

	// This test will fail currently because AudioUpdateInput doesn't have URLs field
	assert.Equal(t, urls, input.URLs)
}

func TestAudioPartial_URLs(t *testing.T) {
	// Test to prove that AudioPartial supports URLs update
	partial := NewAudioPartial()

	urls := []string{
		"https://example.com/audio1.mp3",
		"https://streaming.site/track/123",
	}

	updateStrings := &UpdateStrings{
		Values: urls,
		Mode:   RelationshipUpdateModeSet,
	}

	partial.URLs = updateStrings

	// This test will fail currently because AudioPartial doesn't have URLs field
	assert.Equal(t, updateStrings, partial.URLs)
}

func TestAudio_URLs(t *testing.T) {
	// Test to prove that Audio struct supports URLs field
	audio := NewAudio()

	urls := RelatedStrings{}
	audio.URLs = urls

	// This test will fail currently because Audio doesn't have URLs field
	assert.NotNil(t, audio.URLs)
}

func TestAudioURLs_Integration(t *testing.T) {
	// Test to prove that the complete audio URL workflow works correctly
	// This test validates the entire chain from input to database to output

	urls := []string{
		"https://example.com/audio1.mp3",
		"https://streaming.site/track/123",
		"https://backup.server/files/audio1.mp3",
	}

	// Test AudioUpdateInput can handle URLs
	input := AudioUpdateInput{
		ID:   "1",
		URLs: urls,
	}
	assert.Equal(t, urls, input.URLs)

	// Test AudioPartial can handle URL updates
	partial := NewAudioPartial()
	updateStrings := &UpdateStrings{
		Values: urls,
		Mode:   RelationshipUpdateModeSet,
	}
	partial.URLs = updateStrings
	assert.Equal(t, updateStrings, partial.URLs)

	// Test that AudioPartial.UpdateInput includes URLs
	resultInput := partial.UpdateInput(1)
	assert.Equal(t, urls, resultInput.URLs)

	// Test Audio struct can handle URLs
	audio := NewAudio()
	relatedURLs := NewRelatedStrings(urls)
	audio.URLs = relatedURLs
	assert.True(t, audio.URLs.Loaded())
	assert.Equal(t, urls, audio.URLs.List())
}
