// Package audio provides the application logic for audio files.
// The functionality is exposed via the [Service] type.
package audio

import (
	"context"
	"errors"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

type Service struct {
	File       models.FileReaderWriter
	Repository models.AudioReaderWriter
}

// Update updates an audio with the provided partial data.
func (s *Service) Update(ctx context.Context, id int, input *models.AudioPartial) (*models.Audio, error) {
	return s.Repository.UpdatePartial(ctx, id, *input)
}

// AssignFile assigns a file to an audio.
func (s *Service) AssignFile(ctx context.Context, audioID int, fileID models.FileID) error {
	// ensure file isn't a primary file and that it is an audio file
	f, err := s.File.Find(ctx, fileID)
	if err != nil {
		return err
	}

	ff := f[0]
	if _, ok := ff.(*models.AudioFile); !ok {
		return fmt.Errorf("%s is not an audio file", ff.Base().Path)
	}

	isPrimary, err := s.File.IsPrimary(ctx, fileID)
	if err != nil {
		return err
	}

	if isPrimary {
		return errors.New("cannot reassign primary file")
	}

	return s.Repository.AssignFiles(ctx, audioID, []models.FileID{fileID})
}

// Compile-time check to ensure Service implements the required interface
// This will fail to compile if the Service doesn't implement all required methods
var _ interface {
	Create(ctx context.Context, input *models.Audio, fileIDs []models.FileID) (*models.Audio, error)
	Update(ctx context.Context, id int, input *models.AudioPartial) (*models.Audio, error)
	Destroy(ctx context.Context, audio *models.Audio, fileDeleter *FileDeleter, deleteGenerated, deleteFile bool) error
	AssignFile(ctx context.Context, audioID int, fileID models.FileID) error
	FindByIDs(ctx context.Context, ids []int, load ...LoadRelationshipOption) ([]*models.Audio, error)
} = (*Service)(nil)
