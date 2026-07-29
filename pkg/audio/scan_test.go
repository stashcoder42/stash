package audio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockServerConfig struct{}

func (m *mockServerConfig) GetHost() string              { return "localhost" }
func (m *mockServerConfig) GetPort() int                 { return 9999 }
func (m *mockServerConfig) GetConfigPathAbs() string     { return "/config" }
func (m *mockServerConfig) HasTLSConfig() bool           { return false }
func (m *mockServerConfig) GetPluginsPath() string       { return "/plugins" }
func (m *mockServerConfig) GetDisabledPlugins() []string { return []string{} }
func (m *mockServerConfig) GetPythonPath() string        { return "/usr/bin/python" }

// createTestContext creates a test context with a hook manager to avoid nil pointer issues
func createTestContext() context.Context {
	// For testing, we'll use a context that would cause txn hooks to be no-ops
	// by not having a hook manager
	return context.Background()
}

type mockScanCreatorUpdater struct {
	mock.Mock
}

func (m *mockScanCreatorUpdater) FindByFileID(ctx context.Context, fileID models.FileID) ([]*models.Audio, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0).([]*models.Audio), args.Error(1)
}

func (m *mockScanCreatorUpdater) FindByFingerprints(ctx context.Context, fp []models.Fingerprint) ([]*models.Audio, error) {
	args := m.Called(ctx, fp)
	return args.Get(0).([]*models.Audio), args.Error(1)
}

func (m *mockScanCreatorUpdater) GetFiles(ctx context.Context, relatedID int) ([]models.File, error) {
	args := m.Called(ctx, relatedID)
	return args.Get(0).([]models.File), args.Error(1)
}

func (m *mockScanCreatorUpdater) Create(ctx context.Context, newAudio *models.Audio, fileIDs []models.FileID) error {
	args := m.Called(ctx, newAudio, fileIDs)
	return args.Error(0)
}

func (m *mockScanCreatorUpdater) UpdatePartial(ctx context.Context, id int, updatedAudio models.AudioPartial) (*models.Audio, error) {
	args := m.Called(ctx, id, updatedAudio)
	return args.Get(0).(*models.Audio), args.Error(1)
}

func (m *mockScanCreatorUpdater) AddFileID(ctx context.Context, id int, fileID models.FileID) error {
	args := m.Called(ctx, id, fileID)
	return args.Error(0)
}

type mockScanGenerator struct {
	mock.Mock
}

func (m *mockScanGenerator) Generate(ctx context.Context, a *models.Audio, f *models.AudioFile) error {
	args := m.Called(ctx, a, f)
	return args.Error(0)
}

func makeTestAudioFile() *models.AudioFile {
	return &models.AudioFile{
		BaseFile: &models.BaseFile{
			ID:       1,
			Path:     "/path/to/audio.mp3",
			Basename: "audio.mp3",
			Fingerprints: models.Fingerprints{
				{
					Type:        models.FingerprintTypeMD5,
					Fingerprint: "test-hash",
				},
			},
			Size:      1024,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Duration:   120.5,
		AudioCodec: "mp3",
		Bitrate:    128000,
		SampleRate: 44100,
		Channels:   2,
	}
}

func makeTestAudioForScan() *models.Audio {
	return &models.Audio{
		ID:        1,
		Title:     "Test Audio",
		Path:      "/path/to/audio.mp3",
		Files:     models.NewRelatedFiles([]models.File{makeTestAudioFile()}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestScanHandler_validate(t *testing.T) {
	tests := []struct {
		name    string
		handler *ScanHandler
		wantErr string
	}{
		{
			name:    "nil creator updater",
			handler: &ScanHandler{},
			wantErr: "CreatorUpdater is required",
		},
		{
			name: "nil scan generator",
			handler: &ScanHandler{
				CreatorUpdater: &mockScanCreatorUpdater{},
			},
			wantErr: "ScanGenerator is required",
		},
		{
			name: "invalid file naming algorithm",
			handler: &ScanHandler{
				CreatorUpdater: &mockScanCreatorUpdater{},
				ScanGenerator:  &mockScanGenerator{},
			},
			wantErr: "FileNamingAlgorithm is required",
		},
		{
			name: "nil paths",
			handler: &ScanHandler{
				CreatorUpdater:      &mockScanCreatorUpdater{},
				ScanGenerator:       &mockScanGenerator{},
				FileNamingAlgorithm: models.HashAlgorithmMd5,
			},
			wantErr: "Paths is required",
		},
		{
			name: "valid handler",
			handler: &ScanHandler{
				CreatorUpdater:      &mockScanCreatorUpdater{},
				ScanGenerator:       &mockScanGenerator{},
				FileNamingAlgorithm: models.HashAlgorithmMd5,
				Paths:               &paths.Paths{},
				PluginCache:         plugin.NewCache(&mockServerConfig{}),
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.handler.validate()
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScanHandler_Handle_NotAudioFile(t *testing.T) {
	h := &ScanHandler{
		CreatorUpdater:      &mockScanCreatorUpdater{},
		ScanGenerator:       &mockScanGenerator{},
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
	}

	// Create a non-audio file (BaseFile)
	baseFile := &models.BaseFile{
		ID:   1,
		Path: "/path/to/file.txt",
	}

	err := h.Handle(context.Background(), baseFile, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected audio file but got")
	assert.Contains(t, err.Error(), "/path/to/file.txt")
}

func TestScanHandler_Handle_ValidationError(t *testing.T) {
	h := &ScanHandler{} // Invalid handler

	audioFile := makeTestAudioFile()

	err := h.Handle(context.Background(), audioFile, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CreatorUpdater is required")
}

func TestScanHandler_Handle_CreateNewAudio(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         nil, // Disable plugin cache for tests to avoid transaction hook issues
	}

	audioFile := makeTestAudioFile()
	ctx := createTestContext()

	// Mock finding by file ID - no existing audio
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{}, nil)

	// Mock finding by fingerprints - no existing audio
	mockCreatorUpdater.On("FindByFingerprints", ctx, mock.AnythingOfType("[]models.Fingerprint")).Return([]*models.Audio{}, nil)

	// Mock create - should be called
	mockCreatorUpdater.On("Create", ctx, mock.AnythingOfType("*models.Audio"), []models.FileID{audioFile.ID}).Return(nil)

	// Mock generate - should be called in post-commit hook
	mockGenerator.On("Generate", ctx, mock.AnythingOfType("*models.Audio"), audioFile).Return(nil)

	err := h.Handle(ctx, audioFile, nil)
	assert.NoError(t, err)

	mockCreatorUpdater.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
}

func TestScanHandler_Handle_FindByFileIDError(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         plugin.NewCache(&mockServerConfig{}),
	}

	audioFile := makeTestAudioFile()
	ctx := createTestContext()

	// Mock finding by file ID - error
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{}, errors.New("database error"))

	err := h.Handle(ctx, audioFile, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "finding existing audio")

	mockCreatorUpdater.AssertExpectations(t)
}

func TestScanHandler_Handle_FindByFingerprintsError(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         plugin.NewCache(&mockServerConfig{}),
	}

	audioFile := makeTestAudioFile()
	ctx := createTestContext()

	// Mock finding by file ID - no existing audio
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{}, nil)

	// Mock finding by fingerprints - error
	mockCreatorUpdater.On("FindByFingerprints", ctx, mock.AnythingOfType("[]models.Fingerprint")).Return([]*models.Audio{}, errors.New("database error"))

	err := h.Handle(ctx, audioFile, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "finding existing audio by fingerprints")

	mockCreatorUpdater.AssertExpectations(t)
}

func TestScanHandler_Handle_CreateError(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         plugin.NewCache(&mockServerConfig{}),
	}

	audioFile := makeTestAudioFile()
	ctx := createTestContext()

	// Mock finding by file ID - no existing audio
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{}, nil)

	// Mock finding by fingerprints - no existing audio
	mockCreatorUpdater.On("FindByFingerprints", ctx, mock.AnythingOfType("[]models.Fingerprint")).Return([]*models.Audio{}, nil)

	// Mock create - error
	mockCreatorUpdater.On("Create", ctx, mock.AnythingOfType("*models.Audio"), []models.FileID{audioFile.ID}).Return(errors.New("create error"))

	err := h.Handle(ctx, audioFile, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating new audio")

	mockCreatorUpdater.AssertExpectations(t)
}

func TestScanHandler_Handle_AssociateExisting(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         nil, // Disable plugin cache for tests
	}

	audioFile := makeTestAudioFile()
	existingAudio := makeTestAudioForScan()
	// Pre-populate Files to avoid LoadFiles call
	existingAudio.Files = models.NewRelatedFiles([]models.File{})
	ctx := createTestContext()

	// Mock finding by file ID - existing audio
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{existingAudio}, nil)

	// Mock AddFileID - should be called
	mockCreatorUpdater.On("AddFileID", ctx, existingAudio.ID, audioFile.ID).Return(nil)

	// Mock UpdatePartial - should be called
	mockCreatorUpdater.On("UpdatePartial", ctx, existingAudio.ID, mock.AnythingOfType("models.AudioPartial")).Return(existingAudio, nil)

	// Mock generate - should be called in post-commit hook
	mockGenerator.On("Generate", ctx, existingAudio, audioFile).Return(nil)

	err := h.Handle(ctx, audioFile, nil)
	assert.NoError(t, err)

	mockCreatorUpdater.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
}

func TestScanHandler_Handle_ExistingFileAlreadyAssociated(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         nil, // Disable plugin cache for tests
	}

	audioFile := makeTestAudioFile()
	existingAudio := makeTestAudioForScan()
	// The audio already has this file - pre-populate to avoid LoadFiles call
	existingAudio.Files = models.NewRelatedFiles([]models.File{audioFile})
	ctx := createTestContext()

	// Mock finding by file ID - existing audio
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{existingAudio}, nil)

	// AddFileID and UpdatePartial should NOT be called since file already exists

	// Mock generate - should be called in post-commit hook
	mockGenerator.On("Generate", ctx, existingAudio, audioFile).Return(nil)

	err := h.Handle(ctx, audioFile, nil)
	assert.NoError(t, err)

	mockCreatorUpdater.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
}

func TestScanHandler_Handle_WithOldFile(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         nil, // Disable plugin cache for tests
	}

	audioFile := makeTestAudioFile()
	oldAudioFile := makeTestAudioFile()
	oldAudioFile.ID = 2 // Different ID to make it a different file
	oldAudioFile.Fingerprints[0].Fingerprint = "old-hash"

	existingAudio := makeTestAudioForScan()
	// Pre-populate Files with the old file to avoid LoadFiles call
	existingAudio.Files = models.NewRelatedFiles([]models.File{oldAudioFile})
	ctx := createTestContext()

	// Mock finding by file ID - existing audio
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{existingAudio}, nil)

	// Mock AddFileID - should be called since file changed
	mockCreatorUpdater.On("AddFileID", ctx, existingAudio.ID, audioFile.ID).Return(nil)

	// Mock UpdatePartial - should be called
	mockCreatorUpdater.On("UpdatePartial", ctx, existingAudio.ID, mock.AnythingOfType("models.AudioPartial")).Return(existingAudio, nil)

	// Mock generate - should be called in post-commit hook
	mockGenerator.On("Generate", ctx, existingAudio, audioFile).Return(nil)

	err := h.Handle(ctx, audioFile, oldAudioFile)
	assert.NoError(t, err)

	mockCreatorUpdater.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
}

func TestScanHandler_associateExisting_LoadFilesError(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	h := &ScanHandler{
		CreatorUpdater: mockCreatorUpdater,
	}

	audioFile := makeTestAudioFile()
	existingAudio := makeTestAudioForScan()
	// Clear the Files to force a LoadFiles call
	existingAudio.Files = models.RelatedFiles{}

	// Mock GetFiles to return error when LoadFiles is called
	mockCreatorUpdater.On("GetFiles", mock.Anything, existingAudio.ID).Return([]models.File{}, errors.New("load error"))

	err := h.associateExisting(context.Background(), []*models.Audio{existingAudio}, audioFile, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load error")
}

func TestScanHandler_associateExisting_AddFileIDError(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	h := &ScanHandler{
		CreatorUpdater: mockCreatorUpdater,
	}

	audioFile := makeTestAudioFile()
	existingAudio := makeTestAudioForScan()
	// Pre-populate Files with empty list so the file needs to be added
	existingAudio.Files = models.NewRelatedFiles([]models.File{})

	// Mock AddFileID to return error
	mockCreatorUpdater.On("AddFileID", mock.Anything, existingAudio.ID, audioFile.ID).Return(errors.New("add file error"))

	err := h.associateExisting(context.Background(), []*models.Audio{existingAudio}, audioFile, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "adding file to audio")

	mockCreatorUpdater.AssertExpectations(t)
}

func TestScanHandler_associateExisting_UpdatePartialError(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	h := &ScanHandler{
		CreatorUpdater: mockCreatorUpdater,
	}

	audioFile := makeTestAudioFile()
	existingAudio := makeTestAudioForScan()
	// Pre-populate Files with empty list so the file needs to be added
	existingAudio.Files = models.NewRelatedFiles([]models.File{})

	// Mock AddFileID to succeed
	mockCreatorUpdater.On("AddFileID", mock.Anything, existingAudio.ID, audioFile.ID).Return(nil)

	// Mock UpdatePartial to return error
	mockCreatorUpdater.On("UpdatePartial", mock.Anything, existingAudio.ID, mock.AnythingOfType("models.AudioPartial")).Return((*models.Audio)(nil), errors.New("update error"))

	err := h.associateExisting(context.Background(), []*models.Audio{existingAudio}, audioFile, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "updating audio")

	mockCreatorUpdater.AssertExpectations(t)
}

func TestGetHash(t *testing.T) {
	tests := []struct {
		name        string
		algorithm   models.HashAlgorithm
		fingerprint string
		expected    string
	}{
		{
			name:        "MD5 hash",
			algorithm:   models.HashAlgorithmMd5,
			fingerprint: "test-hash",
			expected:    "test-hash",
		},
		{
			name:        "invalid algorithm",
			algorithm:   "invalid",
			fingerprint: "test-hash",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audioFile := makeTestAudioFile()
			audioFile.Fingerprints[0].Fingerprint = tt.fingerprint

			result := GetHash(audioFile, tt.algorithm)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMigrateHash(t *testing.T) {
	// This is currently a no-op, so just ensure it doesn't panic
	paths := &paths.Paths{}
	MigrateHash(paths, "old-hash", "new-hash")
	// No assertions needed - just ensure it completes
}

func TestScanHandler_Handle_SetsDefaultTitle(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         nil,
	}

	audioFile := &models.AudioFile{
		BaseFile: &models.BaseFile{
			ID:   1,
			Path: "/path/to/My Favorite Song.mp3",
		},
		Duration:   180.5,
		AudioCodec: "mp3",
	}
	ctx := createTestContext()

	// Mock finding by file ID - no existing audio
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{}, nil)

	// Mock finding by fingerprints - no existing audio
	mockCreatorUpdater.On("FindByFingerprints", ctx, mock.AnythingOfType("[]models.Fingerprint")).Return([]*models.Audio{}, nil)

	// Mock create - capture the audio object to verify title is set
	var capturedAudio *models.Audio
	mockCreatorUpdater.On("Create", ctx, mock.AnythingOfType("*models.Audio"), []models.FileID{audioFile.ID}).Run(func(args mock.Arguments) {
		capturedAudio = args.Get(1).(*models.Audio)
	}).Return(nil)

	// Mock generate
	mockGenerator.On("Generate", ctx, mock.AnythingOfType("*models.Audio"), audioFile).Return(nil)

	err := h.Handle(ctx, audioFile, nil)
	assert.NoError(t, err)

	// Verify the title was set to the filename without extension
	assert.NotNil(t, capturedAudio)
	assert.Equal(t, "My Favorite Song", capturedAudio.Title)

	mockCreatorUpdater.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
}

func TestScanHandler_Handle_SetsDefaultTitleFromActualFilename(t *testing.T) {
	mockCreatorUpdater := &mockScanCreatorUpdater{}
	mockGenerator := &mockScanGenerator{}

	h := &ScanHandler{
		CreatorUpdater:      mockCreatorUpdater,
		ScanGenerator:       mockGenerator,
		FileNamingAlgorithm: models.HashAlgorithmMd5,
		Paths:               &paths.Paths{},
		PluginCache:         nil,
	}

	// Test the exact filename from the error message
	audioFile := &models.AudioFile{
		BaseFile: &models.BaseFile{
			ID:   1,
			Path: "/stash/audio/Foobar 15.m4a",
		},
		Duration:   240.0,
		AudioCodec: "aac",
	}
	ctx := createTestContext()

	// Mock finding by file ID - no existing audio
	mockCreatorUpdater.On("FindByFileID", ctx, audioFile.ID).Return([]*models.Audio{}, nil)

	// Mock finding by fingerprints - no existing audio
	mockCreatorUpdater.On("FindByFingerprints", ctx, mock.AnythingOfType("[]models.Fingerprint")).Return([]*models.Audio{}, nil)

	// Mock create - capture the audio object to verify title is set
	var capturedAudio *models.Audio
	mockCreatorUpdater.On("Create", ctx, mock.AnythingOfType("*models.Audio"), []models.FileID{audioFile.ID}).Run(func(args mock.Arguments) {
		capturedAudio = args.Get(1).(*models.Audio)
	}).Return(nil)

	// Mock generate
	mockGenerator.On("Generate", ctx, mock.AnythingOfType("*models.Audio"), audioFile).Return(nil)

	err := h.Handle(ctx, audioFile, nil)
	assert.NoError(t, err)

	// Verify the title was set to the filename without extension
	assert.NotNil(t, capturedAudio)
	assert.Equal(t, "Foobar 15", capturedAudio.Title)
	assert.NotEmpty(t, capturedAudio.Title) // Most importantly, title should not be empty

	mockCreatorUpdater.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
}
