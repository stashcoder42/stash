package audio

import (
	"context"
	"errors"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
)

func TestLoadFiles(t *testing.T) {
	// Setup
	mockReader := &mocks.AudioReaderWriter{}
	audio := &models.Audio{
		ID:    1,
		Files: models.NewRelatedFiles(nil), // Unloaded state
	}
	ctx := context.Background()

	// Test successful loading
	mockReader.On("GetFiles", ctx, 1).Return([]models.File{}, nil).Once()

	err := LoadFiles(ctx, audio, mockReader)

	assert.Nil(t, err)
	mockReader.AssertExpectations(t)

	// Test loading error
	audio2 := &models.Audio{
		ID:    1,
		Files: models.NewRelatedFiles(nil), // Unloaded state
	}
	mockReader.On("GetFiles", ctx, 1).Return(nil, errors.New("db error")).Once()

	err = LoadFiles(ctx, audio2, mockReader)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to load files for audio 1")
	mockReader.AssertExpectations(t)
}

func TestLoadTags(t *testing.T) {
	// Setup
	mockReader := &mocks.AudioReaderWriter{}
	audio := &models.Audio{
		ID:     1,
		TagIDs: models.NewRelatedIDs(nil), // Unloaded state
	}
	ctx := context.Background()

	// Test successful loading
	mockReader.On("GetTagIDs", ctx, 1).Return([]int{}, nil).Once()

	err := LoadTags(ctx, audio, mockReader)

	assert.Nil(t, err)
	mockReader.AssertExpectations(t)

	// Test loading error
	audio2 := &models.Audio{
		ID:     1,
		TagIDs: models.NewRelatedIDs(nil), // Unloaded state
	}
	mockReader.On("GetTagIDs", ctx, 1).Return(nil, errors.New("db error")).Once()

	err = LoadTags(ctx, audio2, mockReader)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to load tag IDs for audio 1")
	mockReader.AssertExpectations(t)
}

func TestLoadPerformers(t *testing.T) {
	// Setup
	mockReader := &mocks.AudioReaderWriter{}
	audio := &models.Audio{
		ID:           1,
		PerformerIDs: models.NewRelatedIDs(nil), // Unloaded state
	}
	ctx := context.Background()

	// Test successful loading
	mockReader.On("GetPerformerIDs", ctx, 1).Return([]int{}, nil).Once()

	err := LoadPerformers(ctx, audio, mockReader)

	assert.Nil(t, err)
	mockReader.AssertExpectations(t)

	// Test loading error
	audio2 := &models.Audio{
		ID:           1,
		PerformerIDs: models.NewRelatedIDs(nil), // Unloaded state
	}
	mockReader.On("GetPerformerIDs", ctx, 1).Return(nil, errors.New("db error")).Once()

	err = LoadPerformers(ctx, audio2, mockReader)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to load performer IDs for audio 1")
	mockReader.AssertExpectations(t)
}

func TestServiceFindByIDs(t *testing.T) {
	// Setup
	mockRepo := &mocks.AudioReaderWriter{}
	service := &Service{Repository: mockRepo}
	ctx := context.Background()
	ids := []int{1, 2}

	// Test successful find
	expectedAudios := []*models.Audio{
		{ID: 1},
		{ID: 2},
	}

	mockRepo.On("FindMany", ctx, ids).Return(expectedAudios, nil).Once()

	audios, err := service.FindByIDs(ctx, ids)

	assert.Nil(t, err)
	assert.Equal(t, expectedAudios, audios)
	mockRepo.AssertExpectations(t)

	// Test find error
	mockRepo.On("FindMany", ctx, ids).Return(nil, errors.New("db error")).Once()

	audios, err = service.FindByIDs(ctx, ids)

	assert.NotNil(t, err)
	assert.Nil(t, audios)
	assert.Contains(t, err.Error(), "db error")
	mockRepo.AssertExpectations(t)
}

func TestServiceFindByIDsWithLoad(t *testing.T) {
	// Setup
	mockRepo := &mocks.AudioReaderWriter{}
	service := &Service{Repository: mockRepo}
	ctx := context.Background()
	ids := []int{1}

	// Create audio without pre-loaded files so LoadFiles will call GetFiles
	audio1 := &models.Audio{
		ID:    1,
		Files: models.NewRelatedFiles(nil), // Unloaded state
	}
	expectedAudios := []*models.Audio{audio1}

	// Mock FindMany
	mockRepo.On("FindMany", ctx, ids).Return(expectedAudios, nil).Once()

	// Mock LoadFiles call for LoadRelationships
	mockRepo.On("GetFiles", ctx, 1).Return([]models.File{}, nil).Once()

	audios, err := service.FindByIDs(ctx, ids, LoadFiles)

	assert.Nil(t, err)
	assert.Equal(t, expectedAudios, audios)
	mockRepo.AssertExpectations(t)
}

func TestServiceFindMany(t *testing.T) {
	// Setup
	mockRepo := &mocks.AudioReaderWriter{}
	service := &Service{Repository: mockRepo}
	ctx := context.Background()
	ids := []int{1, 2}

	// Test successful find
	expectedAudios := []*models.Audio{
		{ID: 1},
		{ID: 2},
	}

	mockRepo.On("FindMany", ctx, ids).Return(expectedAudios, nil).Once()

	audios, err := service.FindMany(ctx, ids)

	assert.Nil(t, err)
	assert.Equal(t, expectedAudios, audios)
	mockRepo.AssertExpectations(t)

	// Test find error
	mockRepo.On("FindMany", ctx, ids).Return(nil, errors.New("db error")).Once()

	audios, err = service.FindMany(ctx, ids)

	assert.NotNil(t, err)
	assert.Nil(t, audios)
	assert.Contains(t, err.Error(), "db error")
	mockRepo.AssertExpectations(t)
}

func TestServiceLoadRelationships(t *testing.T) {
	// Setup
	mockRepo := &mocks.AudioReaderWriter{}
	service := &Service{Repository: mockRepo}
	ctx := context.Background()
	audio := &models.Audio{
		ID:     1,
		Files:  models.NewRelatedFiles(nil), // Unloaded state
		TagIDs: models.NewRelatedIDs(nil),   // Unloaded state
	}

	// Test successful load with multiple loaders
	mockRepo.On("GetFiles", ctx, 1).Return([]models.File{}, nil).Once()
	mockRepo.On("GetTagIDs", ctx, 1).Return([]int{}, nil).Once()

	err := service.LoadRelationships(ctx, audio, LoadFiles, LoadTags)

	assert.Nil(t, err)
	mockRepo.AssertExpectations(t)

	// Test load error
	audio2 := &models.Audio{
		ID:    1,
		Files: models.NewRelatedFiles(nil), // Unloaded state
	}
	mockRepo.On("GetFiles", ctx, 1).Return(nil, errors.New("db error")).Once()

	err = service.LoadRelationships(ctx, audio2, LoadFiles)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to load files for audio 1")
	mockRepo.AssertExpectations(t)
}
