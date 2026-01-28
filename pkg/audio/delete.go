package audio

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

// FileDeleterInterface defines the interface for deleting audio files and generated content
type FileDeleterInterface interface {
	MarkGeneratedFiles(audio *models.Audio) error
	Commit()
	Rollback()
	Files(paths []string) error
}

// FileDeleter is an extension of file.Deleter that handles deletion of audio files.
type FileDeleter struct {
	*file.Deleter

	Paths *paths.Paths
}

// MarkGeneratedFiles marks for deletion the generated files for the provided audio.
func (d *FileDeleter) MarkGeneratedFiles(audio *models.Audio) error {
	var files []string

	// Audio files may have generated content like waveforms, spectrograms, or transcripts
	if audio.Checksum != "" {
		// Check for potential generated files
		// Using similar path structure as images but for audio-specific files

		// Potential waveform image
		waveformPath := d.Paths.Generated.GetThumbnailPath(audio.Checksum, models.DefaultGthumbWidth)
		if exists, _ := fsutil.FileExists(waveformPath); exists {
			files = append(files, waveformPath)
		}

		// Potential spectrogram image
		spectrogramPath := d.Paths.Generated.GetClipPreviewPath(audio.Checksum, models.DefaultGthumbWidth)
		if exists, _ := fsutil.FileExists(spectrogramPath); exists {
			files = append(files, spectrogramPath)
		}

		// Future: transcript files, metadata caches, etc. could be added here
	}

	return d.Files(files)
}

// Destroy destroys an audio, optionally marking the file and generated files for deletion.
func (s *Service) Destroy(ctx context.Context, audio *models.Audio, fileDeleter *FileDeleter, deleteGenerated, deleteFile bool) error {
	return s.destroyAudio(ctx, audio, fileDeleter, deleteGenerated, deleteFile)
}

// DestroyZipAudios destroys all audios in zip, optionally marking the files and generated files for deletion.
// Returns a slice of audios that were destroyed.
// Note: This method currently has limitations due to missing repository methods.
func (s *Service) DestroyZipAudios(ctx context.Context, zipFile models.File, fileDeleter *FileDeleter, deleteGenerated bool) ([]*models.Audio, error) {
	var audiosDestroyed []*models.Audio
	zipFileID := zipFile.Base().ID

	// Find all audios associated with files in this zip
	// Note: This approach finds audios by file ID, but we need a way to find specifically by zip file ID
	// which would require adding FindByZipFileID to the AudioFinder interface
	audios, err := s.Repository.FindByFileID(ctx, zipFileID)
	if err != nil {
		return nil, fmt.Errorf("finding audios by file ID: %w", err)
	}

	for _, audio := range audios {
		if err := audio.LoadFiles(ctx, s.Repository); err != nil {
			return nil, fmt.Errorf("loading files for audio %d: %w", audio.ID, err)
		}

		// If the audio has multiple files, we just want to remove the file in the zip file,
		// not delete the audio entirely
		if len(audio.Files.List()) > 1 {
			// TODO: This requires adding RemoveFileID method to AudioWriter interface
			// For now, we skip multi-file audios in zip deletion
			logger.Infof("Skipping audio %d: has multiple files and RemoveFileID not implemented", audio.ID)
			continue
		}

		// For single-file audios in zip, destroy the entire audio
		const deleteFileInZip = false // Don't delete the actual zip file
		if err := s.destroyAudio(ctx, audio, fileDeleter, deleteGenerated, deleteFileInZip); err != nil {
			return nil, fmt.Errorf("destroying audio %d: %w", audio.ID, err)
		}

		audiosDestroyed = append(audiosDestroyed, audio)
	}

	return audiosDestroyed, nil
}

// DestroyFolderAudios destroys all audios in a folder, optionally marking the files and generated files for deletion.
// Returns a slice of audios that were destroyed.
// Note: This method requires FindByFolderID to be added to AudioFinder interface.
func (s *Service) DestroyFolderAudios(ctx context.Context, folderID models.FolderID, fileDeleter *FileDeleter, deleteGenerated, deleteFile bool) ([]*models.Audio, error) {
	// TODO: Implement when FindByFolderID is added to AudioFinder interface
	// This would follow the same pattern as DestroyFolderImages:
	// 1. Find all audios in the folder
	// 2. For audios with multiple files, only remove files in this folder
	// 3. For audios with single files or no remaining files, destroy the audio

	logger.Warnf("DestroyFolderAudios not fully implemented: requires FindByFolderID method in AudioFinder interface")
	return []*models.Audio{}, nil
}

// destroyAudio destroys an audio, optionally marking the file and generated files for deletion.
func (s *Service) destroyAudio(ctx context.Context, audio *models.Audio, fileDeleter *FileDeleter, deleteGenerated, deleteFile bool) error {
	if deleteFile {
		if err := s.deleteFiles(ctx, audio, fileDeleter); err != nil {
			return fmt.Errorf("deleting files for audio %d: %w", audio.ID, err)
		}
	}

	if deleteGenerated {
		if err := fileDeleter.MarkGeneratedFiles(audio); err != nil {
			return fmt.Errorf("marking generated files for deletion: %w", err)
		}
	}

	if err := s.Repository.Destroy(ctx, audio.ID); err != nil {
		return fmt.Errorf("destroying audio %d from repository: %w", audio.ID, err)
	}

	return nil
}

// deleteFiles deletes files for the audio from the database and file system, if they are not in use by other audios
func (s *Service) deleteFiles(ctx context.Context, audio *models.Audio, fileDeleter *FileDeleter) error {
	if err := audio.LoadFiles(ctx, s.Repository); err != nil {
		return fmt.Errorf("loading files for audio: %w", err)
	}

	for _, f := range audio.Files.List() {
		// Only delete files where there is no other associated audio
		otherAudios, err := s.Repository.FindByFileID(ctx, f.Base().ID)
		if err != nil {
			return fmt.Errorf("checking other audios for file %d: %w", f.Base().ID, err)
		}

		if len(otherAudios) > 1 {
			// Other audio associated, don't remove the file
			logger.Debugf("Skipping deletion of file %d: used by %d audios", f.Base().ID, len(otherAudios))
			continue
		}

		// Don't delete files in zip archives - the zip itself should be managed separately
		if f.Base().ZipFileID == nil {
			logger.Infof("Deleting audio file: %s", f.Base().Path)
			if err := file.Destroy(ctx, s.File, f, fileDeleter.Deleter, true); err != nil {
				return fmt.Errorf("destroying file %s: %w", f.Base().Path, err)
			}
		} else {
			logger.Debugf("Skipping deletion of file %s: part of zip archive", f.Base().Path)
		}
	}

	return nil
}

// DestroyFromInput destroys audios based on GraphQL input.
func (s *Service) DestroyFromInput(ctx context.Context, input models.AudiosDestroyInput, fileDeleter *FileDeleter) error {
	audioIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		return fmt.Errorf("invalid audio ids: %w", err)
	}

	deleteFile := input.DeleteFile != nil && *input.DeleteFile
	deleteGenerated := input.DeleteGenerated != nil && *input.DeleteGenerated

	for _, audioID := range audioIDs {
		audio, err := s.Repository.Find(ctx, audioID)
		if err != nil {
			return fmt.Errorf("error finding audio %d: %w", audioID, err)
		}
		if audio == nil {
			return fmt.Errorf("audio %d not found", audioID)
		}

		if err := s.Destroy(ctx, audio, fileDeleter, deleteGenerated, deleteFile); err != nil {
			return fmt.Errorf("error destroying audio %d: %w", audioID, err)
		}
	}

	return nil
}

// DestroyFromSingleInput destroys a single audio based on GraphQL input.
func (s *Service) DestroyFromSingleInput(ctx context.Context, input models.AudioDestroyInput, fileDeleter *FileDeleter) error {
	audioID, err := stringslice.StringSliceToIntSlice([]string{input.ID})
	if err != nil || len(audioID) != 1 {
		return fmt.Errorf("invalid audio id: %s", input.ID)
	}

	audio, err := s.Repository.Find(ctx, audioID[0])
	if err != nil {
		return fmt.Errorf("error finding audio %d: %w", audioID[0], err)
	}
	if audio == nil {
		return fmt.Errorf("audio %d not found", audioID[0])
	}

	deleteFile := input.DeleteFile != nil && *input.DeleteFile
	deleteGenerated := input.DeleteGenerated != nil && *input.DeleteGenerated

	return s.Destroy(ctx, audio, fileDeleter, deleteGenerated, deleteFile)
}
