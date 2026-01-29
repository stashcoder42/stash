package models

import (
	"time"
)

type AudioMarker struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Seconds      float64   `json:"seconds"`
	EndSeconds   *float64  `json:"end_seconds"`
	PrimaryTagID int       `json:"primary_tag_id"`
	AudioID      int       `json:"audio_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewAudioMarker() AudioMarker {
	currentTime := time.Now()
	return AudioMarker{
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}
}

// AudioMarkerPartial represents part of an AudioMarker object.
// It is used to update the database entry.
type AudioMarkerPartial struct {
	Title        OptionalString
	Seconds      OptionalFloat64
	EndSeconds   OptionalFloat64
	PrimaryTagID OptionalInt
	TagIDs       *UpdateIDs
	AudioID      OptionalInt
	CreatedAt    OptionalTime
	UpdatedAt    OptionalTime
}

func NewAudioMarkerPartial() AudioMarkerPartial {
	currentTime := time.Now()
	return AudioMarkerPartial{
		UpdatedAt: NewOptionalTime(currentTime),
	}
}
