package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

type GenerateAudioThumbnailTask struct {
	repository models.Repository
	Audio      models.Audio
	Overwrite  bool
}

func (t *GenerateAudioThumbnailTask) GetDescription() string {
	return fmt.Sprintf("Generating waveform for %s", t.Audio.GetName())
}

func (t *GenerateAudioThumbnailTask) Start(ctx context.Context) {
	if t.Audio.Path == "" {
		return
	}

	r := t.repository

	var required bool
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		required = t.required(ctx)
		return t.Audio.LoadPrimaryFile(ctx, r.File)
	}); err != nil {
		logger.Error(err)
		return
	}

	if !required {
		logger.Infof("Skipping waveform generation for audio id=%d (already exists)", t.Audio.ID)
		return
	}

	primaryFile := t.Audio.Files.Primary()
	if primaryFile == nil {
		return
	}

	logger.Infof("Generating waveform for audio id=%d path=%s", t.Audio.ID, t.Audio.Path)

	startTime := time.Now()

	// Generate waveform using FFmpeg showwavespic filter
	waveformData, err := t.generateWaveform(ctx, primaryFile.Base().Path)
	if err != nil {
		logger.Errorf("Error generating waveform for audio id=%d: %v", t.Audio.ID, err)
		logErrorOutput(err)
		return
	}

	duration := time.Since(startTime)

	// Store waveform as the audio's cover/thumbnail in the database
	if err := r.WithTxn(ctx, func(ctx context.Context) error {
		qb := r.Audio

		// Update the audio cover table with the waveform image
		if err := qb.UpdateCover(ctx, t.Audio.ID, waveformData); err != nil {
			return fmt.Errorf("error setting waveform: %v", err)
		}

		// Update the audio with the update date
		audioPartial := models.NewAudioPartial()
		_, err := qb.UpdatePartial(ctx, t.Audio.ID, audioPartial)
		if err != nil {
			return fmt.Errorf("error updating audio: %v", err)
		}

		return nil
	}); err != nil && ctx.Err() == nil {
		logger.Error(err.Error())
		return
	}

	logger.Infof("Waveform generated for audio id=%d (size=%d bytes, duration=%.3fs)", t.Audio.ID, len(waveformData), duration.Seconds())
}

func (t *GenerateAudioThumbnailTask) generateWaveform(ctx context.Context, inputPath string) ([]byte, error) {
	// FFmpeg command to generate waveform:
	// ffmpeg -i input.wav -f lavfi -i "color=c=#c0c0c0:s=640x240" -filter_complex \
	//   "[0:a]showwavespic=s=640x240:split_channels=1:colors=#3232c8:filter=peak[pk]; \
	//    [0:a]showwavespic=s=640x240:split_channels=1:colors=#6464dc[rms]; \
	//    [pk][rms]overlay=format=auto[nobg];[1:v][nobg]overlay=format=auto" \
	//   -frames:v 1 -update true output.png

	lockCtx := instance.ReadLockManager.ReadLock(ctx, inputPath)
	defer lockCtx.Cancel()

	// Use a simplified waveform generation that produces a clean visualization
	// Size: 640x360 (16:9), gray background with blue waveform, logarithmic scaling
	// Waveform is 640x324 (90% of height) centered with 18px padding top/bottom
	args := []string{
		"-i", inputPath,
		"-filter_complex",
		"[0:a]showwavespic=s=640x324:colors=#3232c8|#6464dc:split_channels=0:filter=average:scale=log[wave];color=c=#c0c0c0:s=640x360[bg];[bg][wave]overlay=0:18:format=auto",
		"-frames:v", "1",
		"-f", "image2",
		"-y",
		"pipe:1",
	}

	cmd := instance.FFMpeg.Command(ctx, args)

	// Get lock context for input file
	lockCtx.AttachCommand(cmd)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg waveform generation failed: %w", err)
	}

	return output, nil
}

// required returns true if the waveform needs to be generated
func (t *GenerateAudioThumbnailTask) required(ctx context.Context) bool {
	if t.Audio.Path == "" {
		return false
	}

	if t.Overwrite {
		return true
	}

	// Check if the audio already has a cover
	hasCover, err := t.repository.Audio.HasCover(ctx, t.Audio.ID)
	if err != nil {
		logger.Errorf("Error checking audio cover: %v", err)
		return false
	}

	return !hasCover
}

// GenerateAudioWaveformTask generates a waveform image file to disk
// (alternative implementation for generated path storage)
type GenerateAudioWaveformFileTask struct {
	Audio     models.Audio
	Overwrite bool
}

func (t *GenerateAudioWaveformFileTask) GetDescription() string {
	return fmt.Sprintf("Generating waveform file for %s", t.Audio.GetName())
}

func (t *GenerateAudioWaveformFileTask) Start(ctx context.Context) {
	audioChecksum := t.Audio.Checksum

	// Get output path
	thumbPath := instance.Paths.Generated.GetThumbnailPath(audioChecksum, models.DefaultGthumbWidth)

	if !t.required(thumbPath) {
		return
	}

	primaryFile := t.Audio.Files.Primary()
	if primaryFile == nil {
		return
	}

	logger.Debugf("Creating waveform file for %s", t.Audio.Path)

	filePath := primaryFile.Base().Path
	lockCtx := instance.ReadLockManager.ReadLock(ctx, filePath)
	defer lockCtx.Cancel()

	// Ensure parent directory exists
	if err := fsutil.EnsureDirAll(thumbPath); err != nil {
		logger.Errorf("Error creating thumbnail directory: %v", err)
		return
	}

	// Generate waveform to file using FFmpeg
	// Size: 640x360 (16:9), gray background with blue waveform, logarithmic scaling
	// Waveform is 640x324 (90% of height) centered with 18px padding top/bottom
	args := []string{
		"-i", filePath,
		"-filter_complex",
		"[0:a]showwavespic=s=640x324:colors=#3232c8|#6464dc:split_channels=0:filter=average:scale=log[wave];color=c=#c0c0c0:s=640x360[bg];[bg][wave]overlay=0:18:format=auto",
		"-frames:v", "1",
		"-y",
		thumbPath,
	}

	cmd := instance.FFMpeg.Command(ctx, args)
	lockCtx.AttachCommand(cmd)

	if err := cmd.Run(); err != nil {
		logger.Errorf("Error generating waveform file: %v", err)
		logErrorOutput(err)
	}
}

func (t *GenerateAudioWaveformFileTask) required(thumbPath string) bool {
	if t.Audio.Path == "" {
		return false
	}

	if t.Overwrite {
		return true
	}

	exists, _ := fsutil.FileExists(thumbPath)
	return !exists
}
