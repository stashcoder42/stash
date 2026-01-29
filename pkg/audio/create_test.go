package audio

import (
	"context"
	"errors"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testCtx = context.Background()

// makeTestFile creates a test file for mocking
func makeTestFile(id models.FileID) models.File {
	return &models.AudioFile{
		BaseFile: &models.BaseFile{
			ID:       id,
			Path:     "/test/audio.mp3",
			Basename: "audio.mp3",
		},
	}
}

func TestServiceCreate(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	// Test data
	fileID1 := models.FileID(1)
	fileID2 := models.FileID(2)
	fileIDs := []models.FileID{fileID1, fileID2}

	newAudio := models.NewAudio()
	newAudio.Title = "Test Audio"

	// Mock file exists
	mockFileRepo.On("Find", testCtx, fileID1).Return([]models.File{makeTestFile(fileID1)}, nil).Once()
	mockFileRepo.On("Find", testCtx, fileID2).Return([]models.File{makeTestFile(fileID2)}, nil).Once()

	// Mock primary file check
	mockAudioRepo.On("FindByPrimaryFileID", testCtx, fileID1).Return([]*models.Audio{}, nil).Once()

	// Mock create
	mockAudioRepo.On("Create", testCtx, &newAudio, fileIDs).Return(nil).Once()

	createdAudio, err := service.Create(testCtx, &newAudio, fileIDs)

	assert.Nil(t, err)
	assert.NotNil(t, createdAudio)
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceCreateNoFiles_WithTitle_Success(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
	}

	newAudio := models.NewAudio()
	newAudio.Title = "Test Audio Without Files"

	// Mock create with empty files array
	mockAudioRepo.On("Create", testCtx, &newAudio, []models.FileID{}).Return(nil).Once()

	createdAudio, err := service.Create(testCtx, &newAudio, []models.FileID{})

	assert.Nil(t, err)
	assert.NotNil(t, createdAudio)
	mockAudioRepo.AssertExpectations(t)
}

func TestServiceCreateNoFiles_NoTitle_Error(t *testing.T) {
	service := &Service{}

	newAudio := models.NewAudio()
	// No title set

	_, err := service.Create(testCtx, &newAudio, []models.FileID{})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "title must be set if audio has no files")
}

func TestServiceCreateInvalidAudio(t *testing.T) {
	service := &Service{}

	// Create audio with empty title and no files - should fail validation
	newAudio := models.Audio{}

	_, err := service.Create(testCtx, &newAudio, []models.FileID{})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "title must be set if audio has no files")
}

func TestServiceCreateValidAudioNoTitle(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	// Create audio with no title but with files - should be allowed
	newAudio := models.Audio{} // Empty title
	fileID := models.FileID(1)
	fileIDs := []models.FileID{fileID}

	// Mock file exists
	mockFileRepo.On("Find", testCtx, fileID).Return([]models.File{makeTestFile(fileID)}, nil).Once()

	// Mock primary file check
	mockAudioRepo.On("FindByPrimaryFileID", testCtx, fileID).Return([]*models.Audio{}, nil).Once()

	// Mock create
	mockAudioRepo.On("Create", testCtx, &newAudio, fileIDs).Return(nil).Once()

	createdAudio, err := service.Create(testCtx, &newAudio, fileIDs)

	assert.Nil(t, err)
	assert.NotNil(t, createdAudio)
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceCreateFileNotFound(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	fileID := models.FileID(1)
	fileIDs := []models.FileID{fileID}

	newAudio := models.NewAudio()
	newAudio.Title = "Test Audio"

	// Mock file not found
	mockFileRepo.On("Find", testCtx, fileID).Return([]models.File{}, nil).Once()

	_, err := service.Create(testCtx, &newAudio, fileIDs)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "file 1 not found")
	mockFileRepo.AssertExpectations(t)
}

func TestServiceCreateFileAlreadyPrimary(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	fileID := models.FileID(1)
	fileIDs := []models.FileID{fileID}

	newAudio := models.NewAudio()
	newAudio.Title = "Test Audio"

	existingAudio := &models.Audio{ID: 999}

	// Mock file exists
	mockFileRepo.On("Find", testCtx, fileID).Return([]models.File{makeTestFile(fileID)}, nil).Once()

	// Mock file already primary for another audio
	mockAudioRepo.On("FindByPrimaryFileID", testCtx, fileID).Return([]*models.Audio{existingAudio}, nil).Once()

	_, err := service.Create(testCtx, &newAudio, fileIDs)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "file 1 is already primary for another audio")
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceCreateRepositoryError(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	fileID := models.FileID(1)
	fileIDs := []models.FileID{fileID}

	newAudio := models.NewAudio()
	newAudio.Title = "Test Audio"

	// Mock file exists
	mockFileRepo.On("Find", testCtx, fileID).Return([]models.File{makeTestFile(fileID)}, nil).Once()

	// Mock primary file check
	mockAudioRepo.On("FindByPrimaryFileID", testCtx, fileID).Return([]*models.Audio{}, nil).Once()

	// Mock create error
	mockAudioRepo.On("Create", testCtx, &newAudio, fileIDs).Return(errors.New("repository error")).Once()

	_, err := service.Create(testCtx, &newAudio, fileIDs)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error creating audio")
	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceCreateFromInput(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	// Test input
	title := "Test Audio"
	details := "Test details"
	rating := 85
	organized := true
	date := "2023-01-15"
	fileIDStr := "1"

	input := models.AudioCreateInput{
		Title:        &title,
		Details:      &details,
		Rating100:    &rating,
		Organized:    &organized,
		Date:         &date,
		PerformerIds: []string{"1", "2"},
		TagIds:       []string{"3", "4"},
		FileIds:      []string{fileIDStr},
	}

	fileID := models.FileID(1)

	// Mock file exists
	mockFileRepo.On("Find", testCtx, fileID).Return([]models.File{makeTestFile(fileID)}, nil).Once()

	// Mock primary file check
	mockAudioRepo.On("FindByPrimaryFileID", testCtx, fileID).Return([]*models.Audio{}, nil).Once()

	// Mock create
	mockAudioRepo.On("Create", testCtx, mock.AnythingOfType("*models.Audio"), []models.FileID{fileID}).Run(func(args mock.Arguments) {
		audio := args.Get(1).(*models.Audio)
		audio.ID = 123 // Set ID as if created by repository
	}).Return(nil).Once()

	result, err := service.CreateFromInput(testCtx, input)

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, title, result.Title)
	// URLs are not set during creation via AudioCreateInput
	assert.Equal(t, details, result.Details)
	assert.Equal(t, &rating, result.Rating)
	assert.Equal(t, organized, result.Organized)
	assert.Equal(t, []int{1, 2}, result.PerformerIDs.List())
	assert.Equal(t, []int{3, 4}, result.TagIDs.List())

	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestServiceCreateFromInputNoFiles_WithTitle_Success(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
	}

	title := "Test Audio Without Files"
	input := models.AudioCreateInput{
		Title:   &title,
		FileIds: []string{},
	}

	// Mock create with empty files array
	mockAudioRepo.On("Create", testCtx, mock.AnythingOfType("*models.Audio"), []models.FileID{}).Run(func(args mock.Arguments) {
		audio := args.Get(1).(*models.Audio)
		audio.ID = 123
	}).Return(nil).Once()

	result, err := service.CreateFromInput(testCtx, input)

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, title, result.Title)
	mockAudioRepo.AssertExpectations(t)
}

func TestServiceCreateFromInputNoFiles_NoTitle_Error(t *testing.T) {
	service := &Service{}

	// No title, no files - should fail
	input := models.AudioCreateInput{
		FileIds: []string{},
	}

	result, err := service.CreateFromInput(testCtx, input)

	assert.NotNil(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "title must be set if audio has no files")
}

func TestServiceCreateFromInputInvalidDate(t *testing.T) {
	service := &Service{}

	invalidDate := "invalid-date"
	input := models.AudioCreateInput{
		Date:    &invalidDate,
		FileIds: []string{"1"},
	}

	result, err := service.CreateFromInput(testCtx, input)

	assert.NotNil(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestServiceCreateFromInputInvalidPerformerIDs(t *testing.T) {
	service := &Service{}

	input := models.AudioCreateInput{
		PerformerIds: []string{"invalid"},
		FileIds:      []string{"1"},
	}

	result, err := service.CreateFromInput(testCtx, input)

	assert.NotNil(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid performer ids")
}

func TestServiceCreateFromInputInvalidTagIDs(t *testing.T) {
	service := &Service{}

	input := models.AudioCreateInput{
		TagIds:  []string{"invalid"},
		FileIds: []string{"1"},
	}

	result, err := service.CreateFromInput(testCtx, input)

	assert.NotNil(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid tag ids")
}

func TestServiceCreateFromInputInvalidFileIDs(t *testing.T) {
	service := &Service{}

	input := models.AudioCreateInput{
		FileIds: []string{"invalid"},
	}

	result, err := service.CreateFromInput(testCtx, input)

	assert.NotNil(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid file ids")
}

func TestStringSliceToFileIDSlice(t *testing.T) {
	// Test valid conversion
	input := []string{"1", "2", "3"}
	expected := []models.FileID{models.FileID(1), models.FileID(2), models.FileID(3)}

	result, err := stringSliceToFileIDSlice(input)

	assert.Nil(t, err)
	assert.Equal(t, expected, result)

	// Test invalid conversion
	invalidInput := []string{"1", "invalid", "3"}
	result, err = stringSliceToFileIDSlice(invalidInput)

	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestServiceCreateFromInputMinimalData(t *testing.T) {
	mockAudioRepo := &mocks.AudioReaderWriter{}
	mockFileRepo := &mocks.FileReaderWriter{}

	service := &Service{
		Repository: mockAudioRepo,
		File:       mockFileRepo,
	}

	// Minimal input - only file IDs
	input := models.AudioCreateInput{
		FileIds: []string{"1"},
	}

	fileID := models.FileID(1)

	// Mock file exists
	mockFileRepo.On("Find", testCtx, fileID).Return([]models.File{makeTestFile(fileID)}, nil).Once()

	// Mock primary file check
	mockAudioRepo.On("FindByPrimaryFileID", testCtx, fileID).Return([]*models.Audio{}, nil).Once()

	// Mock create
	mockAudioRepo.On("Create", testCtx, mock.AnythingOfType("*models.Audio"), []models.FileID{fileID}).Run(func(args mock.Arguments) {
		audio := args.Get(1).(*models.Audio)
		audio.ID = 123
	}).Return(nil).Once()

	result, err := service.CreateFromInput(testCtx, input)

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "", result.Title) // Should be empty
	// URLs are not set during creation via AudioCreateInput
	assert.Equal(t, 123, result.ID) // Should be set by mock

	mockAudioRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}
