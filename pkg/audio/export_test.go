package audio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/json"
	"github.com/stashapp/stash/pkg/models/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	audioID = 1

	studioID        = 4
	missingStudioID = 5
	errStudioID     = 6

	tagID    = 10
	errTagID = 11

	performerID    = 20
	errPerformerID = 21
)

var (
	title      = "Test Audio Track"
	rating     = 8
	url        = "http://example.com/audio.mp3"
	date       = "2023-01-15"
	dateObj, _ = models.ParseDate(date)
	details    = "A test audio track for testing purposes"
	organized  = true
	ocounter   = 3
)

const (
	studioName    = "Test Studio"
	filePath      = "/path/to/audio.mp3"
	tagName       = "Test Tag"
	performerName = "Test Performer"
)

var (
	createTime = time.Date(2023, 01, 15, 10, 0, 0, 0, time.UTC)
	updateTime = time.Date(2023, 01, 16, 10, 0, 0, 0, time.UTC)
)

func createFullAudio(id int) models.Audio {
	return models.Audio{
		ID: id,
		Files: models.NewRelatedFiles([]models.File{
			&models.BaseFile{
				Path: filePath,
			},
		}),
		Title:        title,
		URLs:         models.NewRelatedStrings([]string{url}),
		Date:         &dateObj,
		Details:      details,
		Rating:       &rating,
		Organized:    organized,
		CreatedAt:    createTime,
		UpdatedAt:    updateTime,
		TagIDs:       models.NewRelatedIDs([]int{tagID}),
		PerformerIDs: models.NewRelatedIDs([]int{performerID}),
	}
}

func createFullJSONAudio() *jsonschema.Audio {
	return &jsonschema.Audio{
		Title:     title,
		URL:       url,
		Date:      date,
		Details:   details,
		Rating:    rating,
		Organized: organized,
		Files:     []string{filePath},
		CreatedAt: json.JSONTime{
			Time: createTime,
		},
		UpdatedAt: json.JSONTime{
			Time: updateTime,
		},
	}
}

func createMinimalAudio(id int) models.Audio {
	return models.Audio{
		ID:           id,
		Title:        title,
		Files:        models.NewRelatedFiles([]models.File{}),
		CreatedAt:    createTime,
		UpdatedAt:    updateTime,
		TagIDs:       models.NewRelatedIDs([]int{}),
		PerformerIDs: models.NewRelatedIDs([]int{}),
	}
}

func createMinimalJSONAudio() *jsonschema.Audio {
	return &jsonschema.Audio{
		Title:     title,
		Organized: false,
		Files:     nil,
		CreatedAt: json.JSONTime{
			Time: createTime,
		},
		UpdatedAt: json.JSONTime{
			Time: updateTime,
		},
	}
}

type basicTestScenario struct {
	input    models.Audio
	expected *jsonschema.Audio
}

var scenarios = []basicTestScenario{
	{
		createFullAudio(audioID),
		createFullJSONAudio(),
	},
	{
		createMinimalAudio(audioID),
		createMinimalJSONAudio(),
	},
}

func TestToBasicJSON(t *testing.T) {
	for i, s := range scenarios {
		audio := s.input
		json := ToBasicJSON(&audio)

		assert.Equal(t, s.expected, json, "[%d]", i)
	}
}

type stringTestScenario struct {
	input    models.Audio
	expected string
	err      bool
}

// Note: Studio tests are placeholder until StudioID is added to Audio model
var getStudioScenarios = []stringTestScenario{
	{
		createFullAudio(audioID),
		"",
		false,
	},
}

func TestGetTagNames(t *testing.T) {
	mockTagReader := &mockTagFinder{}
	audio := createFullAudio(audioID)

	expectedTags := []*models.Tag{
		{ID: tagID, Name: tagName},
	}
	expectedNames := []string{tagName}

	mockTagReader.On("FindByAudioID", testCtx, audioID).Return(expectedTags, nil).Once()

	names, err := GetTagNames(testCtx, mockTagReader, &audio)

	assert.Nil(t, err)
	assert.Equal(t, expectedNames, names)
	mockTagReader.AssertExpectations(t)
}

func TestGetTagNamesError(t *testing.T) {
	mockTagReader := &mockTagFinder{}
	audio := createFullAudio(audioID)

	expectedError := errors.New("tag finder error")
	mockTagReader.On("FindByAudioID", testCtx, audioID).Return(nil, expectedError).Once()

	names, err := GetTagNames(testCtx, mockTagReader, &audio)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error getting audio tags")
	assert.Nil(t, names)
	mockTagReader.AssertExpectations(t)
}

func TestGetPerformerNames(t *testing.T) {
	mockPerformerReader := &mockPerformerFinder{}
	audio := createFullAudio(audioID)

	expectedPerformers := []*models.Performer{
		{ID: performerID, Name: performerName},
	}
	expectedNames := []string{performerName}

	mockPerformerReader.On("FindByAudioID", testCtx, audioID).Return(expectedPerformers, nil).Once()

	names, err := GetPerformerNames(testCtx, mockPerformerReader, &audio)

	assert.Nil(t, err)
	assert.Equal(t, expectedNames, names)
	mockPerformerReader.AssertExpectations(t)
}

func TestGetPerformerNamesError(t *testing.T) {
	mockPerformerReader := &mockPerformerFinder{}
	audio := createFullAudio(audioID)

	expectedError := errors.New("performer finder error")
	mockPerformerReader.On("FindByAudioID", testCtx, audioID).Return(nil, expectedError).Once()

	names, err := GetPerformerNames(testCtx, mockPerformerReader, &audio)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error getting audio performers")
	assert.Nil(t, names)
	mockPerformerReader.AssertExpectations(t)
}

func TestGetDependentTagIDs(t *testing.T) {
	mockTagReader := &mockTagFinder{}
	audio := createFullAudio(audioID)

	expectedTags := []*models.Tag{
		{ID: tagID, Name: tagName},
		{ID: tagID + 1, Name: "Another Tag"},
	}
	expectedIDs := []int{tagID, tagID + 1}

	mockTagReader.On("FindByAudioID", testCtx, audioID).Return(expectedTags, nil).Once()

	ids, err := GetDependentTagIDs(testCtx, mockTagReader, &audio)

	assert.Nil(t, err)
	assert.Equal(t, expectedIDs, ids)
	mockTagReader.AssertExpectations(t)
}

func TestGetDependentTagIDsError(t *testing.T) {
	mockTagReader := &mockTagFinder{}
	audio := createFullAudio(audioID)

	expectedError := errors.New("tag finder error")
	mockTagReader.On("FindByAudioID", testCtx, audioID).Return(nil, expectedError).Once()

	ids, err := GetDependentTagIDs(testCtx, mockTagReader, &audio)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error getting audio tags")
	assert.Nil(t, ids)
	mockTagReader.AssertExpectations(t)
}

func TestGetDependentPerformerIDs(t *testing.T) {
	mockPerformerReader := &mockPerformerFinder{}
	audio := createFullAudio(audioID)

	expectedPerformers := []*models.Performer{
		{ID: performerID, Name: performerName},
		{ID: performerID + 1, Name: "Another Performer"},
	}
	expectedIDs := []int{performerID, performerID + 1}

	mockPerformerReader.On("FindByAudioID", testCtx, audioID).Return(expectedPerformers, nil).Once()

	ids, err := GetDependentPerformerIDs(testCtx, mockPerformerReader, &audio)

	assert.Nil(t, err)
	assert.Equal(t, expectedIDs, ids)
	mockPerformerReader.AssertExpectations(t)
}

func TestGetDependentPerformerIDsError(t *testing.T) {
	mockPerformerReader := &mockPerformerFinder{}
	audio := createFullAudio(audioID)

	expectedError := errors.New("performer finder error")
	mockPerformerReader.On("FindByAudioID", testCtx, audioID).Return(nil, expectedError).Once()

	ids, err := GetDependentPerformerIDs(testCtx, mockPerformerReader, &audio)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error getting audio performers")
	assert.Nil(t, ids)
	mockPerformerReader.AssertExpectations(t)
}

func TestGetTagNamesEmptyList(t *testing.T) {
	tags := []*models.Tag{}
	names := getTagNames(tags)

	assert.Equal(t, []string(nil), names)
}

func TestGetTagNamesFiltersEmpty(t *testing.T) {
	tags := []*models.Tag{
		{ID: 1, Name: "Valid Tag"},
		{ID: 2, Name: ""}, // Empty name should be filtered
		{ID: 3, Name: "Another Valid Tag"},
	}
	expected := []string{"Valid Tag", "Another Valid Tag"}
	names := getTagNames(tags)

	assert.Equal(t, expected, names)
}

func TestGetPerformerNamesEmptyList(t *testing.T) {
	performers := []*models.Performer{}
	names := getPerformerNames(performers)

	assert.Equal(t, []string(nil), names)
}

func TestGetPerformerNamesFiltersEmpty(t *testing.T) {
	performers := []*models.Performer{
		{ID: 1, Name: "Valid Performer"},
		{ID: 2, Name: ""}, // Empty name should be filtered
		{ID: 3, Name: "Another Valid Performer"},
	}
	expected := []string{"Valid Performer", "Another Valid Performer"}
	names := getPerformerNames(performers)

	assert.Equal(t, expected, names)
}

// Test-specific mock implementations since FindByAudioID methods don't exist yet

type mockTagFinder struct {
	mock.Mock
}

func (m *mockTagFinder) FindMany(ctx context.Context, ids []int) ([]*models.Tag, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Tag), args.Error(1)
}

func (m *mockTagFinder) Find(ctx context.Context, id int) (*models.Tag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tag), args.Error(1)
}

func (m *mockTagFinder) FindByAudioID(ctx context.Context, audioID int) ([]*models.Tag, error) {
	args := m.Called(ctx, audioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Tag), args.Error(1)
}

type mockPerformerFinder struct {
	mock.Mock
}

func (m *mockPerformerFinder) FindMany(ctx context.Context, ids []int) ([]*models.Performer, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Performer), args.Error(1)
}

func (m *mockPerformerFinder) Find(ctx context.Context, id int) (*models.Performer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Performer), args.Error(1)
}

func (m *mockPerformerFinder) FindByAudioID(ctx context.Context, audioID int) ([]*models.Performer, error) {
	args := m.Called(ctx, audioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Performer), args.Error(1)
}
