package audio

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

// stringSliceToFileIDSlice converts a slice of strings to a slice of FileIDs.
func stringSliceToFileIDSlice(ss []string) ([]models.FileID, error) {
	ints, err := stringslice.StringSliceToIntSlice(ss)
	if err != nil {
		return nil, err
	}

	ret := make([]models.FileID, len(ints))
	for i, v := range ints {
		ret[i] = models.FileID(v)
	}

	return ret, nil
}

// Create creates a new audio entity with the provided files.
func (s *Service) Create(ctx context.Context, newAudio *models.Audio, fileIDs []models.FileID) (*models.Audio, error) {
	// title must be set if no files are provided (same as scenes)
	if newAudio.Title == "" && len(fileIDs) == 0 {
		return nil, fmt.Errorf("title must be set if audio has no files")
	}

	// Validate rating if present
	if newAudio.Rating != nil && (*newAudio.Rating < 1 || *newAudio.Rating > 100) {
		return nil, fmt.Errorf("rating must be between 1 and 100")
	}

	// Check if files exist and are not already assigned to other audios as primary (only if files provided)
	if len(fileIDs) > 0 {
		for i, fileID := range fileIDs {
			files, err := s.File.Find(ctx, fileID)
			if err != nil {
				return nil, fmt.Errorf("error finding file %d: %w", fileID, err)
			}
			if len(files) == 0 {
				return nil, fmt.Errorf("file %d not found", fileID)
			}

			// Check if file is already primary for another audio (only for first file which becomes primary)
			if i == 0 {
				existingAudios, err := s.Repository.FindByPrimaryFileID(ctx, fileID)
				if err != nil {
					return nil, fmt.Errorf("error checking existing audio for file %d: %w", fileID, err)
				}
				if len(existingAudios) > 0 {
					return nil, fmt.Errorf("file %d is already primary for another audio", fileID)
				}
			}
		}
	}

	// Create the audio
	if err := s.Repository.Create(ctx, newAudio, fileIDs); err != nil {
		return nil, fmt.Errorf("error creating audio: %w", err)
	}

	return newAudio, nil
}

// CreateFromInput creates a new audio from GraphQL input.
func (s *Service) CreateFromInput(ctx context.Context, input models.AudioCreateInput) (*models.Audio, error) {
	newAudio := models.NewAudio()

	// Set basic fields
	if input.Title != nil {
		newAudio.Title = *input.Title
	}

	if input.Details != nil {
		newAudio.Details = *input.Details
	}
	if input.Rating100 != nil {
		newAudio.Rating = input.Rating100
	}
	if input.Organized != nil {
		newAudio.Organized = *input.Organized
	}
	if input.ResumeTime != nil {
		newAudio.ResumeTime = *input.ResumeTime
	}
	if input.PlayDuration != nil {
		newAudio.PlayDuration = *input.PlayDuration
	}

	// Parse date - handle empty strings gracefully like Scene creation
	if input.Date != nil && *input.Date != "" {
		d, err := models.ParseDate(*input.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %w", err)
		}
		newAudio.Date = &d
	}

	// Parse performer IDs
	performerIDs, err := stringslice.StringSliceToIntSlice(input.PerformerIds)
	if err != nil {
		return nil, fmt.Errorf("invalid performer ids: %w", err)
	}
	newAudio.PerformerIDs = models.NewRelatedIDs(performerIDs)

	// Parse tag IDs
	tagIDs, err := stringslice.StringSliceToIntSlice(input.TagIds)
	if err != nil {
		return nil, fmt.Errorf("invalid tag ids: %w", err)
	}
	newAudio.TagIDs = models.NewRelatedIDs(tagIDs)

	// Parse file IDs
	fileIDs, err := stringSliceToFileIDSlice(input.FileIds)
	if err != nil {
		return nil, fmt.Errorf("invalid file ids: %w", err)
	}

	// Create the audio (files are optional now)
	createdAudio, err := s.Create(ctx, &newAudio, fileIDs)
	if err != nil {
		return nil, err
	}

	return createdAudio, nil
}
