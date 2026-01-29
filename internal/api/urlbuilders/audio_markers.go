package urlbuilders

import (
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

type AudioMarkerURLBuilder struct {
	BaseURL  string
	AudioID  string
	MarkerID string
}

func NewAudioMarkerURLBuilder(baseURL string, audioMarker *models.AudioMarker) AudioMarkerURLBuilder {
	return AudioMarkerURLBuilder{
		BaseURL:  baseURL,
		AudioID:  strconv.Itoa(audioMarker.AudioID),
		MarkerID: strconv.Itoa(audioMarker.ID),
	}
}

func (b AudioMarkerURLBuilder) GetStreamURL() string {
	return b.BaseURL + "/audio/" + b.AudioID + "/audio_marker/" + b.MarkerID + "/stream"
}
