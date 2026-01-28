package manager

import (
	"context"
	"net/url"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestGetAudioMimeType(t *testing.T) {
	tests := []struct {
		codec    string
		expected string
	}{
		{"mp3", "audio/mpeg"},
		{"aac", "audio/aac"},
		{"flac", "audio/flac"},
		{"opus", "audio/opus"},
		{"vorbis", "audio/ogg"},
		{"wav", "audio/wav"},
		{"pcm_s16le", "audio/wav"},
		{"alac", "audio/mp4"},
		{"unknown", "audio/mpeg"}, // default fallback
	}

	for _, tt := range tests {
		t.Run(tt.codec, func(t *testing.T) {
			result := GetAudioMimeType(tt.codec)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAudioQualityLabel(t *testing.T) {
	tests := []struct {
		name         string
		audioFile    *models.AudioFile
		endpointType audioEndpointType
		expected     string
	}{
		{
			name: "MP3 with bitrate and stereo",
			audioFile: &models.AudioFile{
				AudioCodec: "mp3",
				Bitrate:    320000,
				SampleRate: 44100,
				Channels:   2,
			},
			endpointType: mp3EndpointType,
			expected:     "MP3 320kbps Stereo",
		},
		{
			name: "FLAC high quality",
			audioFile: &models.AudioFile{
				AudioCodec: "flac",
				Bitrate:    1411000,
				SampleRate: 96000,
				Channels:   2,
			},
			endpointType: audioEndpointType{label: "FLAC"},
			expected:     "FLAC 1411kbps 96kHz Stereo",
		},
		{
			name: "Mono audio",
			audioFile: &models.AudioFile{
				AudioCodec: "aac",
				Bitrate:    128000,
				Channels:   1,
			},
			endpointType: aacEndpointType,
			expected:     "AAC 128kbps Mono",
		},
		{
			name: "Multi-channel audio",
			audioFile: &models.AudioFile{
				AudioCodec: "opus",
				Bitrate:    256000,
				Channels:   6,
			},
			endpointType: audioEndpointType{label: "Opus"},
			expected:     "Opus 256kbps 6ch",
		},
		{
			name:         "No audio file",
			audioFile:    nil,
			endpointType: mp3EndpointType,
			expected:     "MP3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAudioQualityLabel(tt.audioFile, tt.endpointType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDirectAudioStreamable(t *testing.T) {
	tests := []struct {
		name      string
		audioFile *models.AudioFile
		expected  bool
	}{
		{
			name:      "nil audio file",
			audioFile: nil,
			expected:  false,
		},
		{
			name:      "MP3 codec",
			audioFile: &models.AudioFile{AudioCodec: "mp3"},
			expected:  true,
		},
		{
			name:      "AAC codec",
			audioFile: &models.AudioFile{AudioCodec: "aac"},
			expected:  true,
		},
		{
			name:      "FLAC codec",
			audioFile: &models.AudioFile{AudioCodec: "flac"},
			expected:  true,
		},
		{
			name:      "Opus codec",
			audioFile: &models.AudioFile{AudioCodec: "opus"},
			expected:  true,
		},
		{
			name:      "Vorbis codec",
			audioFile: &models.AudioFile{AudioCodec: "vorbis"},
			expected:  true,
		},
		{
			name:      "Unsupported codec",
			audioFile: &models.AudioFile{AudioCodec: "wma"},
			expected:  false,
		},
		{
			name:      "Empty codec",
			audioFile: &models.AudioFile{AudioCodec: ""},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDirectAudioStreamable(tt.audioFile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAudioStreamPaths_BasicScenario(t *testing.T) {
	// Create a test audio with basic file structure
	audio := &models.Audio{
		ID: 1,
		Files: models.NewRelatedFiles([]models.File{
			&models.AudioFile{
				BaseFile: &models.BaseFile{
					ID:   1,
					Path: "/test/audio.mp3",
				},
				AudioCodec: "mp3",
				Bitrate:    320000,
				SampleRate: 44100,
				Channels:   2,
			},
		}),
	}

	streamURL, _ := url.Parse("http://localhost:9999/audio/1/stream")

	// Mock file getter - for this test, we don't need it since Files are already loaded
	mockFileGetter := &MockFileGetter{}

	endpoints, err := GetAudioStreamPaths(context.Background(), audio, streamURL, mockFileGetter)

	assert.NoError(t, err)
	assert.NotEmpty(t, endpoints)

	// Check that we have multiple endpoints for better browser compatibility
	assert.True(t, len(endpoints) > 1, "Should have multiple streaming endpoints")

	// Check the direct stream endpoint (first one)
	directEndpoint := endpoints[0]
	assert.Equal(t, "http://localhost:9999/audio/1/stream", directEndpoint.URL)
	assert.NotNil(t, directEndpoint.MimeType)
	assert.Equal(t, "audio/mpeg", *directEndpoint.MimeType)
	assert.NotNil(t, directEndpoint.Label)
	assert.Contains(t, *directEndpoint.Label, "Direct stream")
	assert.Contains(t, *directEndpoint.Label, "320kbps")
}

// MockFileGetter is a simple mock for testing
type MockFileGetter struct{}

func (m *MockFileGetter) Find(ctx context.Context, ids ...models.FileID) ([]models.File, error) {
	return nil, nil
}
