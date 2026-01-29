package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAudioMarker(t *testing.T) {
	marker := NewAudioMarker()

	assert.NotZero(t, marker.CreatedAt)
	assert.NotZero(t, marker.UpdatedAt)
	assert.Equal(t, marker.CreatedAt, marker.UpdatedAt)

	// Check that the timestamps are recent (within last minute)
	now := time.Now()
	assert.WithinDuration(t, now, marker.CreatedAt, time.Minute)
	assert.WithinDuration(t, now, marker.UpdatedAt, time.Minute)
}

func TestNewAudioMarkerPartial(t *testing.T) {
	partial := NewAudioMarkerPartial()

	assert.True(t, partial.UpdatedAt.Set)
	assert.NotZero(t, partial.UpdatedAt.Value)

	// Check that the timestamp is recent (within last minute)
	now := time.Now()
	assert.WithinDuration(t, now, partial.UpdatedAt.Value, time.Minute)

	// Check that other fields are not set
	assert.False(t, partial.Title.Set)
	assert.False(t, partial.Seconds.Set)
	assert.False(t, partial.EndSeconds.Set)
	assert.False(t, partial.PrimaryTagID.Set)
	assert.False(t, partial.AudioID.Set)
	assert.False(t, partial.CreatedAt.Set)
}

func TestAudioMarkerStruct(t *testing.T) {
	marker := AudioMarker{
		ID:           1,
		Title:        "Test Marker",
		Seconds:      30.5,
		EndSeconds:   nil,
		PrimaryTagID: 10,
		AudioID:      20,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	assert.Equal(t, 1, marker.ID)
	assert.Equal(t, "Test Marker", marker.Title)
	assert.Equal(t, 30.5, marker.Seconds)
	assert.Nil(t, marker.EndSeconds)
	assert.Equal(t, 10, marker.PrimaryTagID)
	assert.Equal(t, 20, marker.AudioID)

	// Test with EndSeconds
	endSeconds := 45.0
	marker.EndSeconds = &endSeconds
	assert.NotNil(t, marker.EndSeconds)
	assert.Equal(t, 45.0, *marker.EndSeconds)
}

func TestAudioMarkerPartialStruct(t *testing.T) {
	partial := AudioMarkerPartial{
		Title:        NewOptionalString("Updated Title"),
		Seconds:      NewOptionalFloat64(60.0),
		PrimaryTagID: NewOptionalInt(15),
		AudioID:      NewOptionalInt(25),
	}

	assert.True(t, partial.Title.Set)
	assert.Equal(t, "Updated Title", partial.Title.Value)
	assert.True(t, partial.Seconds.Set)
	assert.Equal(t, 60.0, partial.Seconds.Value)
	assert.True(t, partial.PrimaryTagID.Set)
	assert.Equal(t, 15, partial.PrimaryTagID.Value)
	assert.True(t, partial.AudioID.Set)
	assert.Equal(t, 25, partial.AudioID.Value)

	// Check that EndSeconds can be set
	endSeconds := 75.0
	partial.EndSeconds = NewOptionalFloat64(endSeconds)
	assert.True(t, partial.EndSeconds.Set)
	assert.Equal(t, endSeconds, partial.EndSeconds.Value)
}
