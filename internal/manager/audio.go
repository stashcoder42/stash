package manager

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/txn"
)

type AudioStreamEndpoint struct {
	URL      string  `json:"url"`
	MimeType *string `json:"mime_type"`
	Label    *string `json:"label"`
}

type audioEndpointType struct {
	label     string
	mimeType  string
	extension string
}

var (
	directAudioEndpointType = audioEndpointType{
		label:     "Direct stream",
		mimeType:  "", // Will be determined dynamically
		extension: "",
	}
	mp3EndpointType = audioEndpointType{
		label:     "MP3",
		mimeType:  "audio/mpeg",
		extension: ".mp3",
	}
	aacEndpointType = audioEndpointType{
		label:     "AAC",
		mimeType:  "audio/aac",
		extension: ".m4a",
	}
	oggEndpointType = audioEndpointType{
		label:     "OGG",
		mimeType:  "audio/ogg",
		extension: ".ogg",
	}
	webmAudioEndpointType = audioEndpointType{
		label:     "WEBM",
		mimeType:  ffmpeg.MimeWebmAudio,
		extension: ".webm",
	}
)

// GetAudioMimeType returns the appropriate MIME type for an audio codec
func GetAudioMimeType(codec string) string {
	switch codec {
	case "mp3":
		return "audio/mpeg"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "opus":
		return "audio/opus"
	case "vorbis":
		return "audio/ogg"
	case "wav", "pcm_s16le", "pcm_s24le", "pcm_s32le":
		return "audio/wav"
	case "alac":
		return "audio/mp4"
	case "ape":
		return "audio/x-ape"
	case "wma":
		return "audio/x-ms-wma"
	default:
		// For unknown codecs, try to make a reasonable guess
		return "audio/mpeg" // Default to MP3 for broad compatibility
	}
}

// GetAudioQualityLabel creates a descriptive label for audio quality
func GetAudioQualityLabel(audioFile *models.AudioFile, endpointType audioEndpointType) string {
	label := endpointType.label

	if audioFile == nil {
		return label
	}

	// Add bitrate information if available
	if audioFile.Bitrate > 0 {
		bitrateKbps := audioFile.Bitrate / 1000
		label += fmt.Sprintf(" %dkbps", bitrateKbps)
	}

	// Add sample rate information for high-quality audio
	if audioFile.SampleRate > 0 {
		if audioFile.SampleRate >= 96000 {
			label += fmt.Sprintf(" %dkHz", audioFile.SampleRate/1000)
		} else if audioFile.SampleRate > 48000 {
			label += fmt.Sprintf(" %.1fkHz", float64(audioFile.SampleRate)/1000)
		}
	}

	// Add channel information for multi-channel audio
	if audioFile.Channels > 2 {
		label += fmt.Sprintf(" %dch", audioFile.Channels)
	} else if audioFile.Channels == 2 {
		label += " Stereo"
	} else if audioFile.Channels == 1 {
		label += " Mono"
	}

	return label
}

// IsDirectAudioStreamable checks if audio can be direct streamed without transcoding
func IsDirectAudioStreamable(audioFile *models.AudioFile) bool {
	if audioFile == nil || audioFile.AudioCodec == "" {
		return false
	}

	// These codecs are generally web-compatible and can be direct streamed
	switch audioFile.AudioCodec {
	case "mp3", "aac", "opus":
		return true
	case "flac":
		// FLAC is supported by modern browsers but may have compatibility issues
		return true
	case "vorbis":
		// Vorbis in OGG container is supported by most browsers
		return true
	default:
		return false
	}
}

// GetAudioStreamPaths generates streaming endpoints for an audio file
func GetAudioStreamPaths(ctx context.Context, audio *models.Audio, directStreamURL *url.URL, fileGetter models.FileGetter) ([]*AudioStreamEndpoint, error) {
	if audio == nil {
		return nil, fmt.Errorf("nil audio")
	}

	// Load the primary file if not already loaded
	if !audio.Files.PrimaryLoaded() {
		if err := audio.LoadPrimaryFile(ctx, fileGetter); err != nil {
			return nil, fmt.Errorf("failed to load primary file: %w", err)
		}
	}

	primaryFile := audio.Files.Primary()
	if primaryFile == nil {
		return nil, nil
	}

	audioFile, ok := primaryFile.(*models.AudioFile)
	if !ok {
		return nil, fmt.Errorf("primary file is not an audio file")
	}

	makeStreamEndpoint := func(t audioEndpointType, audioFile *models.AudioFile) *AudioStreamEndpoint {
		url := *directStreamURL
		url.Path += t.extension

		// Determine MIME type
		mimeType := t.mimeType
		if mimeType == "" {
			// For direct stream, detect MIME type from audio codec
			mimeType = GetAudioMimeType(audioFile.AudioCodec)
		}

		// Create quality label
		label := GetAudioQualityLabel(audioFile, t)

		return &AudioStreamEndpoint{
			URL:      url.String(),
			MimeType: &mimeType,
			Label:    &label,
		}
	}

	var endpoints []*AudioStreamEndpoint

	// Always add direct stream endpoint
	directType := directAudioEndpointType
	directType.mimeType = GetAudioMimeType(audioFile.AudioCodec)
	endpoints = append(endpoints, makeStreamEndpoint(directType, audioFile))

	// Add additional transcoded endpoints based on the source format
	// This provides better browser compatibility

	// Always offer MP3 as it has universal browser support
	if audioFile.AudioCodec != "mp3" {
		endpoints = append(endpoints, makeStreamEndpoint(mp3EndpointType, audioFile))
	}

	// Add AAC/M4A for better quality and Apple device compatibility
	if audioFile.AudioCodec != "aac" {
		endpoints = append(endpoints, makeStreamEndpoint(aacEndpointType, audioFile))
	}

	// Add OGG/Vorbis for open-source compatibility
	if audioFile.AudioCodec != "vorbis" && audioFile.AudioCodec != "opus" {
		endpoints = append(endpoints, makeStreamEndpoint(oggEndpointType, audioFile))
	}

	// Add WebM/Opus for modern browsers
	if audioFile.AudioCodec != "opus" {
		endpoints = append(endpoints, makeStreamEndpoint(webmAudioEndpointType, audioFile))
	}

	return endpoints, nil
}

type AudioServer struct {
	TxnManager txn.Manager
}

func (s *AudioServer) StreamAudioDirect(audio *models.Audio, w http.ResponseWriter, r *http.Request) {
	// Return 404 if the audio does not have a path
	if audio.Path == "" {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	filepath := audio.Path
	streamRequestCtx := ffmpeg.NewStreamRequestContext(w, r)

	// Acquire read lock on the file to prevent deletion during streaming
	_ = GetInstance().ReadLockManager.ReadLock(streamRequestCtx, filepath)
	http.ServeFile(w, r, filepath)
}
