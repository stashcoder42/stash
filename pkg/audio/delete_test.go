package audio

import (
	"errors"
	"testing"

	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stretchr/testify/assert"
)

// makeTestAudio creates a test audio for mocking
func makeTestAudio(id int) *models.Audio {
	return &models.Audio{
		ID:       id,
		Title:    "Test Audio",
		Checksum: "test-checksum",
		Files:    models.NewRelatedFiles([]models.File{makeTestFile(models.FileID(id))}),
	}
}

// makeTestAudioWithoutFiles creates a test audio without pre-loaded files for tests that need to mock GetFiles
func makeTestAudioWithoutFiles(id int) *models.Audio {
	return &models.Audio{
		ID:       id,
		Title:    "Test Audio",
		Checksum: "test-checksum",
		Files:    models.NewRelatedFiles(nil), // No files pre-loaded
	}
}

// makeTestFileDeleter creates a test FileDeleter for mocking
func makeTestFileDeleter() *FileDeleter {
	testPaths := paths.NewPaths("/tmp/test", "/tmp/blobs")
	return &FileDeleter{
		Deleter: file.NewDeleter(),
		Paths:   &testPaths,
	}
}

func TestServiceDestroy(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	audio := makeTestAudio(1)
	fileDeleter := makeTestFileDeleter()

	// Mock repository destroy
	mockAudioRepo.On("Destroy", testCtx, audio.ID).Return(nil).Once()

	err := service.Destroy(testCtx, audio, fileDeleter, false, false)

	assert.Nil(t, err)
	mockAudioRepo.AssertExpectations(t)
}

func TestServiceDestroyWithFileDelete(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	audio := makeTestAudioWithoutFiles(1)
	fileDeleter := makeTestFileDeleter()

	// Mock GetFiles for LoadFiles
	mockAudioRepo.On("GetFiles", testCtx, audio.ID).Return([]models.File{makeTestFile(models.FileID(1))}, nil).Once()

	// Mock FindByFileID (only this audio uses the file)
	mockAudioRepo.On("FindByFileID", testCtx, models.FileID(1)).Return([]*models.Audio{audio}, nil).Once()

	// Mock file destroy
	mockFileRepo.On("Destroy", testCtx, models.FileID(1)).Return(nil).Once()

	// Mock repository destroy
	mockAudioRepo.On("Destroy", testCtx, audio.ID).Return(nil).Once()

	err := service.Destroy(testCtx, audio, fileDeleter, false, true)

	assert.Nil(t, err)
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceDestroyRepositoryError(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	audio := makeTestAudio(1)
	fileDeleter := makeTestFileDeleter()

	// Mock repository destroy error
	mockAudioRepo.On("Destroy", testCtx, audio.ID).Return(errors.New("repository error")).Once()

	err := service.Destroy(testCtx, audio, fileDeleter, false, false)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "destroying audio 1 from repository")
	mockAudioRepo.AssertExpectations(t)
}

func TestServiceDeleteFiles(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	audio := makeTestAudioWithoutFiles(1)
	fileDeleter := makeTestFileDeleter()

	// Mock GetFiles for LoadFiles
	mockAudioRepo.On("GetFiles", testCtx, audio.ID).Return([]models.File{makeTestFile(models.FileID(1))}, nil).Once()

	// Mock FindByFileID (only this audio uses the file)
	mockAudioRepo.On("FindByFileID", testCtx, models.FileID(1)).Return([]*models.Audio{audio}, nil).Once()

	// Mock file destroy
	mockFileRepo.On("Destroy", testCtx, models.FileID(1)).Return(nil).Once()

	err := service.deleteFiles(testCtx, audio, fileDeleter)

	assert.Nil(t, err)
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceDeleteFilesSharedFile(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	audio1 := makeTestAudioWithoutFiles(1)
	audio2 := makeTestAudio(2)
	fileDeleter := makeTestFileDeleter()

	// Mock GetFiles for LoadFiles
	mockAudioRepo.On("GetFiles", testCtx, audio1.ID).Return([]models.File{makeTestFile(models.FileID(1))}, nil).Once()

	// Mock FindByFileID (multiple audios use the file)
	mockAudioRepo.On("FindByFileID", testCtx, models.FileID(1)).Return([]*models.Audio{audio1, audio2}, nil).Once()

	// File should NOT be destroyed since it's shared

	err := service.deleteFiles(testCtx, audio1, fileDeleter)

	assert.Nil(t, err)
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t) // No file operations should occur
}

func TestServiceDeleteFilesInZip(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	audio := makeTestAudioWithoutFiles(1)
	fileDeleter := makeTestFileDeleter()

	// Create a file in zip
	zipFileID := models.FileID(100)
	fileInZip := &models.AudioFile{
		BaseFile: &models.BaseFile{
			ID:       models.FileID(1),
			Path:     "/test/audio.mp3",
			Basename: "audio.mp3",
			DirEntry: models.DirEntry{
				ZipFileID: &zipFileID,
			},
		},
	}

	// Mock GetFiles for LoadFiles
	mockAudioRepo.On("GetFiles", testCtx, audio.ID).Return([]models.File{fileInZip}, nil).Once()

	// Mock FindByFileID (only this audio uses the file)
	mockAudioRepo.On("FindByFileID", testCtx, models.FileID(1)).Return([]*models.Audio{audio}, nil).Once()

	// File should NOT be destroyed since it's in a zip

	err := service.deleteFiles(testCtx, audio, fileDeleter)

	assert.Nil(t, err)
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t) // No file operations should occur
}

func TestServiceDeleteFilesLoadError(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
	}

	audio := makeTestAudioWithoutFiles(1)
	fileDeleter := makeTestFileDeleter()

	// Mock GetFiles error
	mockAudioRepo.On("GetFiles", testCtx, audio.ID).Return(nil, errors.New("load error")).Once()

	err := service.deleteFiles(testCtx, audio, fileDeleter)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "loading files for audio")
	mockAudioRepo.AssertExpectations(t)
}

func TestServiceDestroyFromInput(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	deleteFile := true
	deleteGenerated := true
	input := models.AudiosDestroyInput{
		Ids:             []string{"1", "2"},
		DeleteFile:      &deleteFile,
		DeleteGenerated: &deleteGenerated,
	}

	audio1 := makeTestAudioWithoutFiles(1)
	audio2 := makeTestAudioWithoutFiles(2)
	fileDeleter := makeTestFileDeleter()

	// Mock finds
	mockAudioRepo.On("Find", testCtx, 1).Return(audio1, nil).Once()
	mockAudioRepo.On("Find", testCtx, 2).Return(audio2, nil).Once()

	// Mock GetFiles for both audios
	mockAudioRepo.On("GetFiles", testCtx, 1).Return([]models.File{makeTestFile(models.FileID(1))}, nil).Once()
	mockAudioRepo.On("GetFiles", testCtx, 2).Return([]models.File{makeTestFile(models.FileID(2))}, nil).Once()

	// Mock FindByFileID (each audio uses its own file)
	mockAudioRepo.On("FindByFileID", testCtx, models.FileID(1)).Return([]*models.Audio{audio1}, nil).Once()
	mockAudioRepo.On("FindByFileID", testCtx, models.FileID(2)).Return([]*models.Audio{audio2}, nil).Once()

	// Mock file destroys
	mockFileRepo.On("Destroy", testCtx, models.FileID(1)).Return(nil).Once()
	mockFileRepo.On("Destroy", testCtx, models.FileID(2)).Return(nil).Once()

	// Mock repository destroys
	mockAudioRepo.On("Destroy", testCtx, 1).Return(nil).Once()
	mockAudioRepo.On("Destroy", testCtx, 2).Return(nil).Once()

	err := service.DestroyFromInput(testCtx, input, fileDeleter)

	assert.Nil(t, err)
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceDestroyFromInputInvalidIDs(t *testing.T) {
	service := &Service{}

	input := models.AudiosDestroyInput{
		Ids: []string{"invalid"},
	}

	fileDeleter := makeTestFileDeleter()

	err := service.DestroyFromInput(testCtx, input, fileDeleter)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid audio ids")
}

func TestServiceDestroyFromInputAudioNotFound(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
	}

	input := models.AudiosDestroyInput{
		Ids: []string{"999"},
	}

	fileDeleter := makeTestFileDeleter()

	// Mock find returns nil (not found)
	mockAudioRepo.On("Find", testCtx, 999).Return((*models.Audio)(nil), nil).Once()

	err := service.DestroyFromInput(testCtx, input, fileDeleter)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "audio 999 not found")
	mockAudioRepo.AssertExpectations(t)
}

func TestServiceDestroyFromSingleInput(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	deleteFile := false
	deleteGenerated := true
	input := models.AudioDestroyInput{
		ID:              "1",
		DeleteFile:      &deleteFile,
		DeleteGenerated: &deleteGenerated,
	}

	audio := makeTestAudio(1)
	fileDeleter := makeTestFileDeleter()

	// Mock find
	mockAudioRepo.On("Find", testCtx, 1).Return(audio, nil).Once()

	// Mock repository destroy (no file deletion)
	mockAudioRepo.On("Destroy", testCtx, 1).Return(nil).Once()

	err := service.DestroyFromSingleInput(testCtx, input, fileDeleter)

	assert.Nil(t, err)
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceDestroyFromSingleInputInvalidID(t *testing.T) {
	service := &Service{}

	input := models.AudioDestroyInput{
		ID: "invalid",
	}

	fileDeleter := makeTestFileDeleter()

	err := service.DestroyFromSingleInput(testCtx, input, fileDeleter)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid audio id")
}

func TestServiceDestroyZipAudios(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	zipFileID := models.FileID(100)
	zipFile := &models.AudioFile{
		BaseFile: &models.BaseFile{
			ID:   zipFileID,
			Path: "/test/audio.zip",
		},
	}

	audio := makeTestAudioWithoutFiles(1)
	fileDeleter := makeTestFileDeleter()

	// Mock FindByFileID (finding audios in zip)
	mockAudioRepo.On("FindByFileID", testCtx, zipFileID).Return([]*models.Audio{audio}, nil).Once()

	// Mock GetFiles for LoadFiles
	mockAudioRepo.On("GetFiles", testCtx, audio.ID).Return([]models.File{makeTestFile(models.FileID(1))}, nil).Once()

	// Mock repository destroy (single file audio gets destroyed)
	mockAudioRepo.On("Destroy", testCtx, audio.ID).Return(nil).Once()

	destroyed, err := service.DestroyZipAudios(testCtx, zipFile, fileDeleter, false)

	assert.Nil(t, err)
	assert.Len(t, destroyed, 1)
	assert.Equal(t, audio, destroyed[0])
	mockAudioRepo.AssertExpectations(t)
}

func TestServiceDestroyZipAudiosMultiFile(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
	}

	zipFileID := models.FileID(100)
	zipFile := &models.AudioFile{
		BaseFile: &models.BaseFile{
			ID:   zipFileID,
			Path: "/test/audio.zip",
		},
	}

	// Audio with multiple files - use makeTestAudioWithoutFiles since we expect GetFiles call
	audio := makeTestAudioWithoutFiles(1)

	fileDeleter := makeTestFileDeleter()

	// Mock FindByFileID (finding audios in zip)
	mockAudioRepo.On("FindByFileID", testCtx, zipFileID).Return([]*models.Audio{audio}, nil).Once()

	// Mock GetFiles for LoadFiles
	mockAudioRepo.On("GetFiles", testCtx, audio.ID).Return([]models.File{
		makeTestFile(models.FileID(1)),
		makeTestFile(models.FileID(2)),
	}, nil).Once()

	// Multi-file audio should be skipped, no destroy call

	destroyed, err := service.DestroyZipAudios(testCtx, zipFile, fileDeleter, false)

	assert.Nil(t, err)
	assert.Len(t, destroyed, 0) // Should be skipped
	mockAudioRepo.AssertExpectations(t)
}

func TestServiceDestroyFolderAudios(t *testing.T) {
	service := &Service{}
	fileDeleter := makeTestFileDeleter()

	// This method is not fully implemented yet
	destroyed, err := service.DestroyFolderAudios(testCtx, models.FolderID(1), fileDeleter, false, false)

	assert.Nil(t, err)
	assert.Len(t, destroyed, 0) // Should return empty slice
}

func TestFileDeleterMarkGeneratedFiles(t *testing.T) {
	deleter := makeTestFileDeleter()
	audio := makeTestAudio(1)

	// This test mainly ensures the method doesn't panic
	// Actual file existence checking would require filesystem setup
	err := deleter.MarkGeneratedFiles(audio)

	assert.Nil(t, err)
}

func TestFileDeleterMarkGeneratedFilesEmptyChecksum(t *testing.T) {
	deleter := makeTestFileDeleter()
	audio := &models.Audio{
		ID:       1,
		Title:    "Test Audio",
		Checksum: "", // Empty checksum
	}

	err := deleter.MarkGeneratedFiles(audio)

	assert.Nil(t, err)
}
