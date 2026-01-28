package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stashapp/stash/internal/api/loaders"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAudioResolver_Date(t *testing.T) {
	r := &audioResolver{}

	tests := []struct {
		name     string
		audio    *models.Audio
		expected *string
	}{
		{
			name:     "nil date",
			audio:    &models.Audio{},
			expected: nil,
		},
		{
			name: "valid date",
			audio: &models.Audio{
				Date: &models.Date{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
			expected: func() *string { s := "2023-01-01"; return &s }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Date(context.Background(), tt.audio)
			assert.NoError(t, err)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestAudioResolver_Rating100(t *testing.T) {
	r := &audioResolver{}

	tests := []struct {
		name     string
		audio    *models.Audio
		expected *int
	}{
		{
			name:     "nil rating",
			audio:    &models.Audio{},
			expected: nil,
		},
		{
			name: "valid rating",
			audio: &models.Audio{
				Rating: func() *int { i := 85; return &i }(),
			},
			expected: func() *int { i := 85; return &i }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Rating100(context.Background(), tt.audio)
			assert.NoError(t, err)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestAudioResolver_URL(t *testing.T) {
	// Create a mock database
	db := mocks.NewDatabase()

	// Create a resolver with the mock repository
	r := &audioResolver{
		Resolver: &Resolver{
			repository: db.Repository(),
		},
	}

	tests := []struct {
		name     string
		audio    *models.Audio
		expected *string
	}{
		{
			name:     "no URL",
			audio:    &models.Audio{URLs: models.NewRelatedStrings([]string{})},
			expected: nil,
		},
		{
			name: "single URL",
			audio: &models.Audio{
				URLs: models.NewRelatedStrings([]string{"http://example.com/audio.mp3"}),
			},
			expected: func() *string { s := "http://example.com/audio.mp3"; return &s }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.URL(context.Background(), tt.audio)
			assert.NoError(t, err)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestAudioResolver_Urls(t *testing.T) {
	// Create a mock database
	db := mocks.NewDatabase()

	// Create a resolver with the mock repository
	r := &audioResolver{
		Resolver: &Resolver{
			repository: db.Repository(),
		},
	}

	tests := []struct {
		name     string
		audio    *models.Audio
		expected []string
	}{
		{
			name:     "no URL",
			audio:    &models.Audio{URLs: models.NewRelatedStrings([]string{})},
			expected: []string{},
		},
		{
			name: "single URL",
			audio: &models.Audio{
				URLs: models.NewRelatedStrings([]string{"http://example.com/audio.mp3"}),
			},
			expected: []string{"http://example.com/audio.mp3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Urls(context.Background(), tt.audio)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAudioResolver_Paths(t *testing.T) {
	// Create a mock database
	db := mocks.NewDatabase()

	// Create a resolver with the mock repository
	r := &audioResolver{
		Resolver: &Resolver{
			repository: db.Repository(),
		},
	}

	audio := &models.Audio{
		ID:        1,
		Checksum:  "abc123",
		UpdatedAt: time.Unix(1234567890, 0),
	}

	// Mock the HasCover method that the Paths method calls
	db.Audio.On("HasCover", mock.Anything, 1).Return(true, nil).Once()

	baseURL := "http://localhost:9999"
	ctx := context.WithValue(context.Background(), BaseURLCtxKey, baseURL)

	result, err := r.Paths(ctx, audio)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Stream)
	assert.NotNil(t, result.Cover)
	assert.NotNil(t, result.Preview)

	assert.Contains(t, *result.Stream, "/audio/1/stream")
	assert.Contains(t, *result.Cover, "/audio/1/cover")
	assert.Contains(t, *result.Preview, "/audio/1/thumbnail")

	// Verify that the expected repository calls were made
	db.AssertExpectations(t)
}

// Helper function to create a context with real loaders backed by mock repository
func createTestContextWithMockRepository(db *mocks.Database) context.Context {
	// Create a fetch function that simulates the middleware's fetch behavior
	fetchFunc := func(keys []int) ([]int, []error) {
		result, err := db.Repository().Audio.GetManyViewCount(context.Background(), keys)
		if err != nil {
			return nil, []error{err}
		}
		return result, nil
	}

	// Create a fetch function for LastPlayed
	fetchLastPlayedFunc := func(keys []int) ([]*time.Time, []error) {
		result, err := db.Repository().Audio.GetManyLastViewed(context.Background(), keys)
		if err != nil {
			return nil, []error{err}
		}
		return result, nil
	}

	// Create a fetch function for PlayHistory
	fetchPlayHistoryFunc := func(keys []int) ([][]time.Time, []error) {
		result, err := db.Repository().Audio.GetManyViewDates(context.Background(), keys)
		if err != nil {
			return nil, []error{err}
		}
		return result, nil
	}

	// Create a fetch function for OHistory
	fetchOHistoryFunc := func(keys []int) ([][]time.Time, []error) {
		result, err := db.Repository().Audio.GetManyODates(context.Background(), keys)
		if err != nil {
			return nil, []error{err}
		}
		return result, nil
	}

	// Create real loaders using the mock repository
	audioPlayCountLoader := loaders.NewAudioPlayCountLoader(loaders.AudioPlayCountLoaderConfig{
		Fetch:    fetchFunc,
		Wait:     1 * time.Millisecond,
		MaxBatch: 100,
	})

	audioLastPlayedLoader := loaders.NewAudioLastPlayedLoader(loaders.AudioLastPlayedLoaderConfig{
		Fetch:    fetchLastPlayedFunc,
		Wait:     1 * time.Millisecond,
		MaxBatch: 100,
	})

	audioPlayHistoryLoader := loaders.NewAudioPlayHistoryLoader(loaders.AudioPlayHistoryLoaderConfig{
		Fetch:    fetchPlayHistoryFunc,
		Wait:     1 * time.Millisecond,
		MaxBatch: 100,
	})

	audioOHistoryLoader := loaders.NewAudioOHistoryLoader(loaders.AudioOHistoryLoaderConfig{
		Fetch:    fetchOHistoryFunc,
		Wait:     1 * time.Millisecond,
		MaxBatch: 100,
	})

	// Create the full loaders struct (only need audio loaders for these tests)
	testLoaders := loaders.Loaders{
		AudioPlayCount:   audioPlayCountLoader,
		AudioLastPlayed:  audioLastPlayedLoader,
		AudioPlayHistory: audioPlayHistoryLoader,
		AudioOHistory:    audioOHistoryLoader,
	}

	// Use the ForTest helper function to create a context with the exact same key
	return loaders.ForTest(context.Background(), testLoaders)
}

func TestAudioResolver_PlayCount(t *testing.T) {
	r := &audioResolver{}

	tests := []struct {
		name          string
		audioID       int
		expectedCount int
		expectError   bool
		repoError     error
	}{
		{
			name:          "valid play count",
			audioID:       1,
			expectedCount: 5,
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "zero play count",
			audioID:       2,
			expectedCount: 0,
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "repository error",
			audioID:       3,
			expectedCount: 0,
			expectError:   true,
			repoError:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db := mocks.NewDatabase()

			// Mock the GetManyViewCount method that the loader calls
			if tt.repoError != nil {
				db.Audio.On("GetManyViewCount", mock.Anything, []int{tt.audioID}).
					Return([]int{}, tt.repoError).Once()
			} else {
				db.Audio.On("GetManyViewCount", mock.Anything, []int{tt.audioID}).
					Return([]int{tt.expectedCount}, nil).Once()
			}

			// Create context with real loaders backed by mock repository
			ctx := createTestContextWithMockRepository(db)

			// Create test audio
			audio := &models.Audio{ID: tt.audioID}

			// Call the method
			result, err := r.PlayCount(ctx, audio)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedCount, *result)
			}

			// Verify that the expected repository calls were made
			db.AssertExpectations(t)
		})
	}
}

func TestAudioResolver_LastPlayedAt(t *testing.T) {
	r := &audioResolver{}

	testTime := time.Date(2023, 12, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		audioID      int
		expectedTime *time.Time
		expectError  bool
		repoError    error
	}{
		{
			name:         "valid last played time",
			audioID:      1,
			expectedTime: &testTime,
			expectError:  false,
			repoError:    nil,
		},
		{
			name:         "nil last played time (never played)",
			audioID:      2,
			expectedTime: nil,
			expectError:  false,
			repoError:    nil,
		},
		{
			name:         "repository error",
			audioID:      3,
			expectedTime: nil,
			expectError:  true,
			repoError:    errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db := mocks.NewDatabase()

			// Mock the GetManyLastViewed method that the loader calls
			if tt.repoError != nil {
				db.Audio.On("GetManyLastViewed", mock.Anything, []int{tt.audioID}).
					Return([]*time.Time{}, tt.repoError).Once()
			} else {
				db.Audio.On("GetManyLastViewed", mock.Anything, []int{tt.audioID}).
					Return([]*time.Time{tt.expectedTime}, nil).Once()
			}

			// Create context with real loaders backed by mock repository
			ctx := createTestContextWithMockRepository(db)

			// Create test audio
			audio := &models.Audio{ID: tt.audioID}

			// Call the method
			result, err := r.LastPlayedAt(ctx, audio)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.expectedTime == nil {
					assert.Nil(t, result)
				} else {
					assert.NotNil(t, result)
					assert.Equal(t, tt.expectedTime.UTC(), result.UTC())
				}
			}

			// Verify that the expected repository calls were made
			db.AssertExpectations(t)
		})
	}
}

func TestAudioResolver_PlayHistory(t *testing.T) {
	r := &audioResolver{}

	time1 := time.Date(2023, 12, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 12, 2, 12, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 12, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		audioID       int
		expectedTimes []time.Time
		expectError   bool
		repoError     error
	}{
		{
			name:          "multiple play dates",
			audioID:       1,
			expectedTimes: []time.Time{time1, time2, time3},
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "single play date",
			audioID:       2,
			expectedTimes: []time.Time{time1},
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "no play history (empty slice)",
			audioID:       3,
			expectedTimes: []time.Time{},
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "repository error",
			audioID:       4,
			expectedTimes: nil,
			expectError:   true,
			repoError:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db := mocks.NewDatabase()

			// Mock the GetManyViewDates method that the loader calls
			if tt.repoError != nil {
				db.Audio.On("GetManyViewDates", mock.Anything, []int{tt.audioID}).
					Return([][]time.Time{}, tt.repoError).Once()
			} else {
				db.Audio.On("GetManyViewDates", mock.Anything, []int{tt.audioID}).
					Return([][]time.Time{tt.expectedTimes}, nil).Once()
			}

			// Create context with real loaders backed by mock repository
			ctx := createTestContextWithMockRepository(db)

			// Create test audio
			audio := &models.Audio{ID: tt.audioID}

			// Call the method
			result, err := r.PlayHistory(ctx, audio)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, len(tt.expectedTimes), len(result))

				// Verify each timestamp
				for i, expectedTime := range tt.expectedTimes {
					assert.NotNil(t, result[i])
					assert.Equal(t, expectedTime.UTC(), result[i].UTC())
				}
			}

			// Verify that the expected repository calls were made
			db.AssertExpectations(t)
		})
	}
}

func TestAudioResolver_OHistory(t *testing.T) {
	r := &audioResolver{}

	time1 := time.Date(2023, 12, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 12, 2, 12, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 12, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		audioID       int
		expectedTimes []time.Time
		expectError   bool
		repoError     error
	}{
		{
			name:          "multiple O dates",
			audioID:       1,
			expectedTimes: []time.Time{time1, time2, time3},
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "single O date",
			audioID:       2,
			expectedTimes: []time.Time{time1},
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "no O history (empty slice)",
			audioID:       3,
			expectedTimes: []time.Time{},
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "repository error",
			audioID:       4,
			expectedTimes: nil,
			expectError:   true,
			repoError:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db := mocks.NewDatabase()

			// Mock the GetManyODates method that the loader calls
			if tt.repoError != nil {
				db.Audio.On("GetManyODates", mock.Anything, []int{tt.audioID}).
					Return([][]time.Time{}, tt.repoError).Once()
			} else {
				db.Audio.On("GetManyODates", mock.Anything, []int{tt.audioID}).
					Return([][]time.Time{tt.expectedTimes}, nil).Once()
			}

			// Create context with real loaders backed by mock repository
			ctx := createTestContextWithMockRepository(db)

			// Create test audio
			audio := &models.Audio{ID: tt.audioID}

			// Call the method
			result, err := r.OHistory(ctx, audio)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, len(tt.expectedTimes), len(result))

				// Verify each timestamp
				for i, expectedTime := range tt.expectedTimes {
					assert.NotNil(t, result[i])
					assert.Equal(t, expectedTime.UTC(), result[i].UTC())
				}
			}

			// Verify that the expected repository calls were made
			db.AssertExpectations(t)
		})
	}
}

func TestAudioResolver_OCounter(t *testing.T) {
	r := &audioResolver{}

	tests := []struct {
		name          string
		audioID       int
		expectedCount int
		expectError   bool
		repoError     error
	}{
		{
			name:          "valid O count",
			audioID:       1,
			expectedCount: 3,
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "zero O count",
			audioID:       2,
			expectedCount: 0,
			expectError:   false,
			repoError:     nil,
		},
		{
			name:          "repository error",
			audioID:       3,
			expectedCount: 0,
			expectError:   true,
			repoError:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db := mocks.NewDatabase()

			// Mock the GetManyOCount method that the loader calls
			if tt.repoError != nil {
				db.Audio.On("GetManyOCount", mock.Anything, []int{tt.audioID}).
					Return([]int{}, tt.repoError).Once()
			} else {
				db.Audio.On("GetManyOCount", mock.Anything, []int{tt.audioID}).
					Return([]int{tt.expectedCount}, nil).Once()
			}

			// Create a fetch function for OCount
			fetchOCountFunc := func(keys []int) ([]int, []error) {
				result, err := db.Repository().Audio.GetManyOCount(context.Background(), keys)
				if err != nil {
					return nil, []error{err}
				}
				return result, nil
			}

			// Create O count loader
			audioOCountLoader := loaders.NewAudioOCountLoader(loaders.AudioOCountLoaderConfig{
				Fetch:    fetchOCountFunc,
				Wait:     1 * time.Millisecond,
				MaxBatch: 100,
			})

			// Create context with the loader
			testLoaders := loaders.Loaders{
				AudioOCount: audioOCountLoader,
			}
			ctx := loaders.ForTest(context.Background(), testLoaders)

			// Create test audio
			audio := &models.Audio{ID: tt.audioID}

			// Call the method
			result, err := r.OCounter(ctx, audio)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedCount, *result)
			}

			// Verify that the expected repository calls were made
			db.AssertExpectations(t)
		})
	}
}
