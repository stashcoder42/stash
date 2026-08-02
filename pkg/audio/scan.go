package audio

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/txn"
)

// hookManagerCtx returns the hook manager from context, or nil if not found
func hookManagerCtx(ctx context.Context) interface{} {
	// Use the same key type and value as txn package
	type key int
	const hookManagerKey key = iota + 1
	return ctx.Value(hookManagerKey)
}

var (
	// fingerprint types to match with
	// only try to match by data fingerprints, _not_ perceptual fingerprints
	matchableFingerprintTypes = []string{models.FingerprintTypeMD5}
)

type ScanCreatorUpdater interface {
	FindByFileID(ctx context.Context, fileID models.FileID) ([]*models.Audio, error)
	FindByFingerprints(ctx context.Context, fp []models.Fingerprint) ([]*models.Audio, error)
	GetFiles(ctx context.Context, relatedID int) ([]models.File, error)

	Create(ctx context.Context, newAudio *models.Audio, fileIDs []models.FileID) error
	UpdatePartial(ctx context.Context, id int, updatedAudio models.AudioPartial) (*models.Audio, error)
	AddFileID(ctx context.Context, id int, fileID models.FileID) error
}

type ScanGenerator interface {
	Generate(ctx context.Context, a *models.Audio, f *models.AudioFile) error
}

type ScanHandler struct {
	CreatorUpdater ScanCreatorUpdater

	ScanGenerator ScanGenerator
	PluginCache   *plugin.Cache

	FileNamingAlgorithm models.HashAlgorithm
	Paths               *paths.Paths
}

func (h *ScanHandler) validate() error {
	if h.CreatorUpdater == nil {
		return errors.New("CreatorUpdater is required")
	}
	if h.ScanGenerator == nil {
		return errors.New("ScanGenerator is required")
	}
	if !h.FileNamingAlgorithm.IsValid() {
		return errors.New("FileNamingAlgorithm is required")
	}
	if h.Paths == nil {
		return errors.New("internal error: Paths is required")
	}

	return nil
}

func (h *ScanHandler) Handle(ctx context.Context, f models.File, oldFile models.File) error {
	if err := h.validate(); err != nil {
		return err
	}

	audioFile, ok := f.(*models.AudioFile)
	if !ok {
		return fmt.Errorf("expected audio file but got %T for file: %s", f, f.Base().Path)
	}

	// try to match the file to an audio
	existing, err := h.CreatorUpdater.FindByFileID(ctx, f.Base().ID)
	if err != nil {
		return fmt.Errorf("finding existing audio: %w", err)
	}

	if len(existing) == 0 {
		// try also to match file by fingerprints
		existing, err = h.CreatorUpdater.FindByFingerprints(ctx, audioFile.Fingerprints.Filter(matchableFingerprintTypes...))
		if err != nil {
			return fmt.Errorf("finding existing audio by fingerprints: %w", err)
		}
	}

	if len(existing) > 0 {
		updateExisting := oldFile != nil
		if err := h.associateExisting(ctx, existing, audioFile, updateExisting); err != nil {
			return err
		}
	} else {
		// create a new audio
		newAudio := models.NewAudio()

		// Set default title to filename without extension if no title is provided
		if newAudio.Title == "" {
			filename := filepath.Base(audioFile.Path)
			// Remove file extension for cleaner title
			if ext := filepath.Ext(filename); ext != "" {
				filename = filename[:len(filename)-len(ext)]
			}
			newAudio.Title = filename
		}

		logger.Infof("%s doesn't exist. Creating new audio...", f.Base().Path)

		if err := h.CreatorUpdater.Create(ctx, &newAudio, []models.FileID{audioFile.ID}); err != nil {
			return fmt.Errorf("creating new audio: %w", err)
		}

		if h.PluginCache != nil {
			h.PluginCache.RegisterPostHooks(ctx, newAudio.ID, hook.AudioCreatePost, nil, nil)
		}

		existing = []*models.Audio{&newAudio}
	}

	if oldFile != nil {
		// migrate hashes from the old file to the new
		oldHash := GetHash(oldFile, h.FileNamingAlgorithm)
		newHash := GetHash(f, h.FileNamingAlgorithm)

		if oldHash != "" && newHash != "" && oldHash != newHash {
			MigrateHash(h.Paths, oldHash, newHash)
		}
	}

	// do this after the commit so that generation doesn't hold up the transaction
	// Check if we have a valid transaction context before adding hooks
	if hookManagerCtx(ctx) != nil {
		txn.AddPostCommitHook(ctx, func(ctx context.Context) {
			for _, a := range existing {
				if err := h.ScanGenerator.Generate(ctx, a, audioFile); err != nil {
					// just log if generation fails. We can try again on rescan
					logger.Errorf("Error generating content for %s: %v", audioFile.Path, err)
				}
			}
		})
	} else {
		// For testing or contexts without transaction hooks, generate immediately
		for _, a := range existing {
			if err := h.ScanGenerator.Generate(ctx, a, audioFile); err != nil {
				// just log if generation fails. We can try again on rescan
				logger.Errorf("Error generating content for %s: %v", audioFile.Path, err)
			}
		}
	}

	return nil
}

func (h *ScanHandler) associateExisting(ctx context.Context, existing []*models.Audio, f *models.AudioFile, updateExisting bool) error {
	for _, a := range existing {
		if err := a.LoadFiles(ctx, h.CreatorUpdater); err != nil {
			return err
		}

		found := false
		for _, af := range a.Files.List() {
			if af.Base().ID == f.Base().ID {
				found = true
				break
			}
		}

		if !found {
			logger.Infof("Adding %s to audio %s", f.Path, a.DisplayName())

			if err := h.CreatorUpdater.AddFileID(ctx, a.ID, f.ID); err != nil {
				return fmt.Errorf("adding file to audio: %w", err)
			}

			// update updated_at time
			audioPartial := models.NewAudioPartial()
			if _, err := h.CreatorUpdater.UpdatePartial(ctx, a.ID, audioPartial); err != nil {
				return fmt.Errorf("updating audio: %w", err)
			}
		}

		if !found || updateExisting {
			if h.PluginCache != nil {
				h.PluginCache.RegisterPostHooks(ctx, a.ID, hook.AudioUpdatePost, nil, nil)
			}
		}
	}

	return nil
}

// GetHash returns a hash for the given file and algorithm.
func GetHash(f models.File, algorithm models.HashAlgorithm) string {
	switch algorithm {
	case models.HashAlgorithmMd5:
		return f.Base().Fingerprints.GetString(models.FingerprintTypeMD5)
	default:
		return ""
	}
}

// MigrateHash migrates generated files from one hash to another.
func MigrateHash(paths *paths.Paths, oldHash, newHash string) {
	// Currently a no-op for audio files since we don't generate previews/thumbnails yet
	// This can be extended when audio preview generation is implemented
	logger.Debugf("Audio hash migration from %s to %s (no-op)", oldHash, newHash)
}
