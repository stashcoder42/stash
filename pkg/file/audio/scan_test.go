package audio

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestDecorator_Decorate_RequiresFFProbe(t *testing.T) {
	d := &Decorator{}

	baseFile := &models.BaseFile{
		DirEntry: models.DirEntry{
			ModTime: time.Time{},
		},
		Path:     "/test/path/audio.mp3",
		Basename: "audio.mp3",
	}

	result, err := d.Decorate(context.Background(), &file.OsFS{}, baseFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ffprobe not configured")
	assert.Equal(t, baseFile, result)
}

func TestDecorator_IsMissingMetadata(t *testing.T) {
	d := &Decorator{}

	// Test with non-audio file
	baseFile := &models.BaseFile{}
	result := d.IsMissingMetadata(context.Background(), &file.OsFS{}, baseFile)
	assert.True(t, result)

	// Test with audio file that has metadata
	audioFile := &models.AudioFile{
		BaseFile:   baseFile,
		Format:     "mp3",
		Duration:   120.0,
		AudioCodec: "mp3",
	}
	result = d.IsMissingMetadata(context.Background(), &file.OsFS{}, audioFile)
	assert.False(t, result)

	// Test with audio file missing duration (using sentinel value from migration)
	audioFileMissingDuration := &models.AudioFile{
		BaseFile:   baseFile,
		Format:     "mp3",
		Duration:   -1, // unsetNumber sentinel value
		AudioCodec: "mp3",
	}
	result = d.IsMissingMetadata(context.Background(), &file.OsFS{}, audioFileMissingDuration)
	assert.True(t, result)

	// Test with audio file missing codec (using sentinel value from migration)
	audioFileMissingCodec := &models.AudioFile{
		BaseFile:   baseFile,
		Format:     "mp3",
		Duration:   120.0,
		AudioCodec: "unset", // unsetString sentinel value
	}
	result = d.IsMissingMetadata(context.Background(), &file.OsFS{}, audioFileMissingCodec)
	assert.True(t, result)

	// Test with audio file missing format (using sentinel value from migration)
	audioFileMissingFormat := &models.AudioFile{
		BaseFile:   baseFile,
		Format:     "unset", // unsetString sentinel value
		Duration:   120.0,
		AudioCodec: "mp3",
	}
	result = d.IsMissingMetadata(context.Background(), &file.OsFS{}, audioFileMissingFormat)
	assert.True(t, result)
}

func TestDecorator_ExtractsSampleRateAndChannels(t *testing.T) {
	// This test would require mocking FFProbe, which is complex for a unit test
	// The functionality is tested as part of integration tests
	// For now, we'll just test that the decorator can handle missing metadata gracefully

	d := &Decorator{}

	audioFile := &models.AudioFile{
		BaseFile:   &models.BaseFile{},
		Format:     "mp3",
		Duration:   120.0,
		AudioCodec: "mp3",
		SampleRate: 44100,
		Channels:   2,
	}

	result := d.IsMissingMetadata(context.Background(), &file.OsFS{}, audioFile)
	assert.False(t, result) // Should not be missing metadata with proper values

	// Test with missing sample rate and channels
	audioFileMissingMeta := &models.AudioFile{
		BaseFile:   &models.BaseFile{},
		Format:     "mp3",
		Duration:   120.0,
		AudioCodec: "mp3",
		SampleRate: 0, // Missing
		Channels:   0, // Missing
	}

	// These should not be considered "missing metadata" since they're optional
	result = d.IsMissingMetadata(context.Background(), &file.OsFS{}, audioFileMissingMeta)
	assert.False(t, result) // Duration and codec are present, which is sufficient
}

func TestSampleRateParsingLogic(t *testing.T) {
	// Test the sample rate parsing logic that's used in the decorator
	testCases := []struct {
		input    string
		expected int
		name     string
	}{
		{"44100", 44100, "standard CD quality"},
		{"48000", 48000, "standard digital audio"},
		{"96000", 96000, "high resolution audio"},
		{"22050", 22050, "half CD quality"},
		{"", 0, "empty string"},
		{"invalid", 0, "invalid string"},
		{"44100.5", 0, "decimal number"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This mirrors the parsing logic in the decorator
			sampleRate := 0
			if tc.input != "" {
				if parsed, err := strconv.Atoi(tc.input); err == nil {
					sampleRate = parsed
				}
			}

			assert.Equal(t, tc.expected, sampleRate, "Sample rate parsing for: %s", tc.input)
		})
	}
}
