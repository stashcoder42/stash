package urlbuilders

import (
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestNewAudioURLBuilder(t *testing.T) {
	baseURL := "http://localhost:9999"
	audio := &models.Audio{
		ID:        123,
		Checksum:  "abc123def456",
		UpdatedAt: time.Unix(1234567890, 0),
	}

	builder := NewAudioURLBuilder(baseURL, audio)

	assert.Equal(t, baseURL, builder.BaseURL)
	assert.Equal(t, "123", builder.AudioID)
	assert.Equal(t, "abc123def456", builder.Checksum)
	assert.Equal(t, "1234567890", builder.UpdatedAt)
}

func TestAudioURLBuilder_GetStreamURL(t *testing.T) {
	builder := AudioURLBuilder{
		BaseURL:   "http://localhost:9999",
		AudioID:   "123",
		Checksum:  "abc123def456",
		UpdatedAt: "1234567890",
	}

	tests := []struct {
		name        string
		apiKey      string
		expectedURL string
	}{
		{
			name:        "without API key",
			apiKey:      "",
			expectedURL: "http://localhost:9999/audio/123/stream",
		},
		{
			name:        "with API key",
			apiKey:      "test-api-key",
			expectedURL: "http://localhost:9999/audio/123/stream?apikey=test-api-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := builder.GetStreamURL(tt.apiKey)
			assert.Equal(t, tt.expectedURL, url.String())
		})
	}
}

func TestAudioURLBuilder_GetCoverURL(t *testing.T) {
	builder := AudioURLBuilder{
		BaseURL:   "http://localhost:9999",
		AudioID:   "123",
		Checksum:  "abc123def456",
		UpdatedAt: "1234567890",
	}

	expected := "http://localhost:9999/audio/123/cover?t=1234567890"
	result := builder.GetCoverURL()

	assert.Equal(t, expected, result)
}

func TestAudioURLBuilder_GetThumbnailURL(t *testing.T) {
	builder := AudioURLBuilder{
		BaseURL:   "http://localhost:9999",
		AudioID:   "123",
		Checksum:  "abc123def456",
		UpdatedAt: "1234567890",
	}

	expected := "http://localhost:9999/audio/123/thumbnail?t=1234567890"
	result := builder.GetThumbnailURL()

	assert.Equal(t, expected, result)
}

func TestAudioURLBuilderGetCaptionPath(t *testing.T) {
	b := AudioURLBuilder{
		BaseURL: "http://localhost:9999",
		AudioID: "123",
	}

	const want = "/audio/123/caption"
	if got := b.GetCaptionPath(); got != want {
		t.Errorf("GetCaptionPath() = %q, want %q", got, want)
	}

	const wantURL = "http://localhost:9999/audio/123/caption"
	if got := b.GetCaptionURL(); got != wantURL {
		t.Errorf("GetCaptionURL() = %q, want %q", got, wantURL)
	}

	// GetCaptionURL must always equal BaseURL + GetCaptionPath()
	if got, want := b.GetCaptionURL(), b.BaseURL+b.GetCaptionPath(); got != want {
		t.Errorf("GetCaptionURL() = %q, want BaseURL+GetCaptionPath() = %q", got, want)
	}
}

func TestAudioURLBuilder_GetStreamURL_PanicOnInvalidURL(t *testing.T) {
	// Create a builder with an invalid base URL that would cause url.Parse to fail
	builder := AudioURLBuilder{
		BaseURL:   ":", // Invalid URL
		AudioID:   "123",
		Checksum:  "abc123def456",
		UpdatedAt: "1234567890",
	}

	assert.Panics(t, func() {
		builder.GetStreamURL("")
	})
}

func TestAudioURLBuilder_WithEmptyValues(t *testing.T) {
	builder := AudioURLBuilder{
		BaseURL:   "",
		AudioID:   "",
		Checksum:  "",
		UpdatedAt: "",
	}

	// Test that methods don't panic with empty values
	assert.NotPanics(t, func() {
		result := builder.GetCoverURL()
		assert.Equal(t, "/audio//cover?t=", result)
	})

	assert.NotPanics(t, func() {
		result := builder.GetThumbnailURL()
		assert.Equal(t, "/audio//thumbnail?t=", result)
	})

	// GetStreamURL should still work with empty base URL (though the result might not be useful)
	assert.NotPanics(t, func() {
		builder.GetStreamURL("")
	})
}

func TestAudioURLBuilder_WithSpecialCharacters(t *testing.T) {
	builder := AudioURLBuilder{
		BaseURL:   "http://localhost:9999",
		AudioID:   "123",
		Checksum:  "abc123def456",
		UpdatedAt: "1234567890",
	}

	// Test with API key containing special characters
	apiKey := "test-api-key-with-special-chars!@#$%^&*()"
	url := builder.GetStreamURL(apiKey)

	// The URL should properly encode the API key
	assert.Contains(t, url.String(), "apikey=")
	assert.Contains(t, url.RawQuery, "test-api-key-with-special-chars")
}
