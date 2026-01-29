package api

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test constants
const (
	testAudioIDConst       = 1
	testAudio2IDConst      = 2
	testInvalidAudioID     = 999
	testAudioChecksumConst = "abcd1234"
)

// Test data
var (
	testAudioForQuery = &models.Audio{
		ID:        testAudioIDConst,
		Title:     "Test Audio",
		Checksum:  testAudioChecksumConst,
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	testAudio2ForQuery = &models.Audio{
		ID:        testAudio2IDConst,
		Title:     "Test Audio 2",
		Checksum:  "efgh5678",
		CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	testAudioFileForQuery = &models.AudioFile{
		BaseFile: &models.BaseFile{
			ID:   1,
			Size: 1024000,
		},
		Duration: 180.5,
	}
)

// newAudioTestResolver creates a resolver with mock database for audio tests
func newAudioTestResolver(db *mocks.Database) *Resolver {
	return &Resolver{
		repository:   db.Repository(),
		hookExecutor: &mockHookExecutor{},
	}
}

// =============================================================================
// FindAudio Query Tests
// =============================================================================

func TestQueryResolver_FindAudio(t *testing.T) {
	tests := []struct {
		name       string
		id         *string
		checksum   *string
		setupMocks func(db *mocks.Database)
		expected   *models.Audio
		wantErr    bool
	}{
		{
			name: "find by valid ID",
			id:   func() *string { s := strconv.Itoa(testAudioIDConst); return &s }(),
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("Find", mock.Anything, testAudioIDConst).Return(testAudioForQuery, nil)
			},
			expected: testAudioForQuery,
			wantErr:  false,
		},
		{
			name: "find by invalid ID returns nil",
			id:   func() *string { s := strconv.Itoa(testInvalidAudioID); return &s }(),
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("Find", mock.Anything, testInvalidAudioID).Return(nil, nil)
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "find by valid checksum",
			checksum: func() *string { s := testAudioChecksumConst; return &s }(),
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("FindByChecksum", mock.Anything, testAudioChecksumConst).Return([]*models.Audio{testAudioForQuery}, nil)
			},
			expected: testAudioForQuery,
			wantErr:  false,
		},
		{
			name:     "find by invalid checksum returns nil",
			checksum: func() *string { s := "invalid"; return &s }(),
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("FindByChecksum", mock.Anything, "invalid").Return([]*models.Audio{}, nil)
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "invalid ID format returns error",
			id:   func() *string { s := "not-a-number"; return &s }(),
			setupMocks: func(db *mocks.Database) {
				// No mock needed - should error before calling repo
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "no parameters returns nil",
			setupMocks: func(db *mocks.Database) {
				// No mock needed
			},
			expected: nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Query().FindAudio(context.Background(), tt.id, tt.checksum)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// =============================================================================
// FindAudios Query Tests
// =============================================================================

func TestQueryResolver_FindAudios(t *testing.T) {
	// Create copies for tests that modify file loading
	testAudioCopy := *testAudioForQuery
	testAudio2Copy := *testAudio2ForQuery
	testAudioCopy.Files = models.NewRelatedFiles([]models.File{testAudioFileForQuery})
	testAudio2Copy.Files = models.NewRelatedFiles([]models.File{testAudioFileForQuery})

	audioList := []*models.Audio{&testAudioCopy, &testAudio2Copy}
	audioIDs := []int{testAudioIDConst, testAudio2IDConst}
	audioIDStrings := []string{strconv.Itoa(testAudioIDConst), strconv.Itoa(testAudio2IDConst)}

	tests := []struct {
		name        string
		ids         []string
		audioFilter *models.AudioFilterType
		filter      *models.FindFilterType
		setupMocks  func(db *mocks.Database)
		expectCount int
		wantErr     bool
	}{
		{
			name: "find by string IDs",
			ids:  audioIDStrings,
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("FindMany", mock.Anything, audioIDs).Return(audioList, nil)
				db.File.On("Find", mock.Anything, mock.Anything).Return([]models.File{testAudioFileForQuery}, nil).Maybe()
			},
			expectCount: 2,
			wantErr:     false,
		},
		{
			name: "find with title filter",
			audioFilter: &models.AudioFilterType{
				Title: &models.StringCriterionInput{
					Value:    "Test",
					Modifier: models.CriterionModifierEquals,
				},
			},
			setupMocks: func(db *mocks.Database) {
				queryResult := models.NewAudioQueryResult(db.Audio)
				queryResult.Count = 2
				queryResult.TotalDuration = 361.0
				queryResult.TotalSize = 2048000.0
				queryResult.IDs = audioIDs
				db.Audio.On("Query", mock.Anything, mock.AnythingOfType("models.AudioQueryOptions")).Return(queryResult, nil)
				db.Audio.On("FindMany", mock.Anything, audioIDs).Return(audioList, nil)
			},
			expectCount: 2,
			wantErr:     false,
		},
		{
			name: "invalid string ID returns error",
			ids:  []string{"not-a-number"},
			setupMocks: func(db *mocks.Database) {
				// No mock needed - should error before calling repo
			},
			expectCount: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Query().FindAudios(context.Background(), tt.audioFilter, tt.ids, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectCount, result.Count)
				if tt.expectCount > 0 {
					assert.Len(t, result.Audios, tt.expectCount)
				}
			}
		})
	}
}

// =============================================================================
// Audio O-Counter Mutation Tests
// =============================================================================

func TestMutationResolver_AudioIncrementO(t *testing.T) {
	tests := []struct {
		name       string
		audioID    string
		setupMocks func(db *mocks.Database)
		expected   int
		wantErr    bool
	}{
		{
			name:    "increment o-counter successfully",
			audioID: strconv.Itoa(testAudioIDConst),
			setupMocks: func(db *mocks.Database) {
				// AddO returns the updated times list
				db.Audio.On("AddO", mock.Anything, testAudioIDConst, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name:    "invalid ID returns error",
			audioID: "not-a-number",
			setupMocks: func(db *mocks.Database) {
				// No mock needed
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Mutation().AudioIncrementO(context.Background(), tt.audioID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMutationResolver_AudioDecrementO(t *testing.T) {
	tests := []struct {
		name       string
		audioID    string
		setupMocks func(db *mocks.Database)
		expected   int
		wantErr    bool
	}{
		{
			name:    "decrement o-counter successfully",
			audioID: strconv.Itoa(testAudioIDConst),
			setupMocks: func(db *mocks.Database) {
				// DeleteO removes the most recent entry, returns remaining
				db.Audio.On("DeleteO", mock.Anything, testAudioIDConst, mock.Anything).
					Return([]time.Time{}, nil)
			},
			expected: 0,
			wantErr:  false,
		},
		{
			name:    "invalid ID returns error",
			audioID: "not-a-number",
			setupMocks: func(db *mocks.Database) {
				// No mock needed
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Mutation().AudioDecrementO(context.Background(), tt.audioID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMutationResolver_AudioResetO(t *testing.T) {
	tests := []struct {
		name       string
		audioID    string
		setupMocks func(db *mocks.Database)
		expected   int
		wantErr    bool
	}{
		{
			name:    "reset o-counter successfully",
			audioID: strconv.Itoa(testAudioIDConst),
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("ResetO", mock.Anything, testAudioIDConst).Return(0, nil)
			},
			expected: 0,
			wantErr:  false,
		},
		{
			name:    "invalid ID returns error",
			audioID: "not-a-number",
			setupMocks: func(db *mocks.Database) {
				// No mock needed
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Mutation().AudioResetO(context.Background(), tt.audioID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMutationResolver_AudioAddO(t *testing.T) {
	testTime1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	testTime2 := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		audioID    string
		times      []*time.Time
		setupMocks func(db *mocks.Database)
		expected   int
		wantErr    bool
	}{
		{
			name:    "add specific timestamps",
			audioID: strconv.Itoa(testAudioIDConst),
			times:   []*time.Time{&testTime1, &testTime2},
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("Find", mock.Anything, testAudioIDConst).
					Return(&models.Audio{ID: testAudioIDConst}, nil)
				db.Audio.On("AddO", mock.Anything, testAudioIDConst, mock.AnythingOfType("[]time.Time")).
					Return([]time.Time{testTime1, testTime2}, nil)
			},
			expected: 2,
			wantErr:  false,
		},
		{
			name:    "add without times (adds current time)",
			audioID: strconv.Itoa(testAudioIDConst),
			times:   nil,
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("Find", mock.Anything, testAudioIDConst).
					Return(&models.Audio{ID: testAudioIDConst}, nil)
				db.Audio.On("AddO", mock.Anything, testAudioIDConst, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name:    "invalid ID returns error",
			audioID: "not-a-number",
			times:   nil,
			setupMocks: func(db *mocks.Database) {
				// No mock needed
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Mutation().AudioAddO(context.Background(), tt.audioID, tt.times)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, result.Count)
			}
		})
	}
}

func TestMutationResolver_AudioDeleteO(t *testing.T) {
	testTime1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		audioID    string
		times      []*time.Time
		setupMocks func(db *mocks.Database)
		expected   int
		wantErr    bool
	}{
		{
			name:    "delete specific timestamps",
			audioID: strconv.Itoa(testAudioIDConst),
			times:   []*time.Time{&testTime1},
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("Find", mock.Anything, testAudioIDConst).
					Return(&models.Audio{ID: testAudioIDConst}, nil)
				db.Audio.On("DeleteO", mock.Anything, testAudioIDConst, mock.AnythingOfType("[]time.Time")).
					Return([]time.Time{}, nil)
			},
			expected: 0,
			wantErr:  false,
		},
		{
			name:    "delete most recent (no times specified)",
			audioID: strconv.Itoa(testAudioIDConst),
			times:   nil,
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("Find", mock.Anything, testAudioIDConst).
					Return(&models.Audio{ID: testAudioIDConst}, nil)
				db.Audio.On("DeleteO", mock.Anything, testAudioIDConst, mock.Anything).
					Return([]time.Time{testTime1}, nil)
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name:    "invalid ID returns error",
			audioID: "not-a-number",
			times:   nil,
			setupMocks: func(db *mocks.Database) {
				// No mock needed
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Mutation().AudioDeleteO(context.Background(), tt.audioID, tt.times)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, result.Count)
			}
		})
	}
}

// =============================================================================
// Audio Play Count Mutation Tests
// =============================================================================

func TestMutationResolver_AudioIncrementPlayCount(t *testing.T) {
	tests := []struct {
		name       string
		audioID    string
		setupMocks func(db *mocks.Database)
		expected   int
		wantErr    bool
	}{
		{
			name:    "increment play count successfully",
			audioID: strconv.Itoa(testAudioIDConst),
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("AddViews", mock.Anything, testAudioIDConst, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
			},
			expected: 1,
			wantErr:  false,
		},
		{
			name:    "invalid ID returns error",
			audioID: "not-a-number",
			setupMocks: func(db *mocks.Database) {
				// No mock needed
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Mutation().AudioIncrementPlayCount(context.Background(), tt.audioID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMutationResolver_AudioResetPlayCount(t *testing.T) {
	tests := []struct {
		name       string
		audioID    string
		setupMocks func(db *mocks.Database)
		expected   int
		wantErr    bool
	}{
		{
			name:    "reset play count successfully",
			audioID: strconv.Itoa(testAudioIDConst),
			setupMocks: func(db *mocks.Database) {
				db.Audio.On("DeleteAllViews", mock.Anything, testAudioIDConst).Return(0, nil)
			},
			expected: 0,
			wantErr:  false,
		},
		{
			name:    "invalid ID returns error",
			audioID: "not-a-number",
			setupMocks: func(db *mocks.Database) {
				// No mock needed
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			r := newAudioTestResolver(db)

			tt.setupMocks(db)

			result, err := r.Mutation().AudioResetPlayCount(context.Background(), tt.audioID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
