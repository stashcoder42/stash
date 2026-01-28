package audio

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/models"
)

// Decorator adds audio specific fields to a File.
type Decorator struct {
	FFProbe *ffmpeg.FFProbe
}

func (d *Decorator) Decorate(ctx context.Context, fs models.FS, f models.File) (models.File, error) {
	if d.FFProbe == nil {
		return f, errors.New("ffprobe not configured")
	}

	base := f.Base()
	// TODO - copy to temp file if not an OsFS
	if _, isOs := fs.(*file.OsFS); !isOs {
		return f, fmt.Errorf("audio.Decorator: only OsFS is supported")
	}

	probe := d.FFProbe
	videoFile, err := probe.NewVideoFile(base.Path)
	if err != nil {
		return f, fmt.Errorf("running ffprobe on audio file %q: %w", base.Path, err)
	}

	// Extract audio-specific information
	audioCodec := ""
	if videoFile.AudioStream != nil {
		audioCodec = videoFile.AudioStream.CodecName
	}

	// Extract additional audio metadata
	bitrate := videoFile.Bitrate
	sampleRate := 0
	channels := 0

	if videoFile.AudioStream != nil {
		// Parse sample rate from string to int
		if videoFile.AudioStream.SampleRate != "" {
			if parsed, err := strconv.Atoi(videoFile.AudioStream.SampleRate); err == nil {
				sampleRate = parsed
			}
		}

		// Extract channels directly (already an int)
		channels = videoFile.AudioStream.Channels
	}

	// Extract format/container information like VideoFile does
	container, err := ffmpeg.MatchContainer(videoFile.Container, base.Path)
	if err != nil {
		return f, fmt.Errorf("matching container for %q: %w", base.Path, err)
	}

	audioFile := &models.AudioFile{
		BaseFile:   base,
		Format:     string(container),
		Duration:   videoFile.FileDuration,
		AudioCodec: audioCodec,
		Bitrate:    bitrate,
		SampleRate: sampleRate,
		Channels:   channels,
	}

	return audioFile, nil
}

func (d *Decorator) IsMissingMetadata(ctx context.Context, fs models.FS, f models.File) bool {
	const (
		unsetString = "unset"
		unsetNumber = -1
	)

	audioFile, isAudio := f.(*models.AudioFile)
	if !isAudio {
		return true
	}

	// Check if essential audio metadata is missing (following VideoFile pattern)
	return audioFile.AudioCodec == unsetString || audioFile.Format == unsetString ||
		audioFile.Duration == unsetNumber
}
