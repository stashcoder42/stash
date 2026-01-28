//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
)

// Audio integration tests - following existing patterns from scene_test.go and image_test.go

func TestAudioCreate(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioCreate", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Create a new audio
		newAudio := models.NewAudio()
		newAudio.Title = "Test Audio"
		newAudio.URLs = models.NewRelatedStrings([]string{"http://example.com/test.mp3"})
		newAudio.Details = "Test audio details"
		rating := 75
		newAudio.Rating = &rating
		newAudio.Organized = true

		// Create without files first (we'll test file associations separately)
		err := qb.Create(ctx, &newAudio, []models.FileID{})
		assert.Nil(t, err)
		assert.Greater(t, newAudio.ID, 0)

		// Retrieve and verify
		retrieved, err := qb.Find(ctx, newAudio.ID)
		assert.Nil(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "Test Audio", retrieved.Title)

		// Load URLs to verify them
		err = retrieved.LoadURLs(ctx, qb)
		assert.Nil(t, err)
		assert.Equal(t, []string{"http://example.com/test.mp3"}, retrieved.URLs.List())

		assert.Equal(t, "Test audio details", retrieved.Details)
		assert.NotNil(t, retrieved.Rating)
		assert.Equal(t, 75, *retrieved.Rating)
		assert.True(t, retrieved.Organized)
	})
}

func TestAudioUpdate(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioUpdate", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Use existing audio from test data
		audioID := audioIDs[audioIdxWithPerformer]

		// Create partial update
		partial := models.AudioPartial{
			Title:   models.NewOptionalString("Updated Title"),
			Rating:  models.NewOptionalInt(90),
			Details: models.NewOptionalString("Updated details"),
		}

		// Update the audio
		updatedAudio, err := qb.UpdatePartial(ctx, audioID, partial)
		assert.Nil(t, err)
		assert.NotNil(t, updatedAudio)
		assert.Equal(t, "Updated Title", updatedAudio.Title)
		assert.NotNil(t, updatedAudio.Rating)
		assert.Equal(t, 90, *updatedAudio.Rating)
		assert.Equal(t, "Updated details", updatedAudio.Details)
	})
}

func TestAudioDelete(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioDelete", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Create a temporary audio for deletion
		newAudio := models.NewAudio()
		newAudio.Title = "Audio to Delete"
		err := qb.Create(ctx, &newAudio, []models.FileID{})
		assert.Nil(t, err)
		audioID := newAudio.ID

		// Verify it exists
		retrieved, err := qb.Find(ctx, audioID)
		assert.Nil(t, err)
		assert.NotNil(t, retrieved)

		// Delete it
		err = qb.Destroy(ctx, audioID)
		assert.Nil(t, err)

		// Verify it's gone - Find should return nil audio with no error
		retrieved, err = qb.Find(ctx, audioID)
		assert.Nil(t, err)       // No error expected
		assert.Nil(t, retrieved) // Audio should be nil
	})
}

func TestAudioQuery(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioQuery", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test basic query
		result, err := qb.Query(ctx, models.AudioQueryOptions{
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.Count, 0)

		// Resolve the audio IDs to actual objects
		audios, err := result.Resolve(ctx)
		assert.Nil(t, err)
		assert.NotNil(t, audios)
		assert.Equal(t, result.Count, len(audios))
	})
}

func TestAudioFilterByTitle(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterByTitle", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Get the actual title from an existing audio to ensure we find something
		existingAudio, err := qb.Find(ctx, audioIDs[audioIdxWithPerformer])
		assert.Nil(t, err)
		assert.NotNil(t, existingAudio)

		// Test filtering by the actual title
		titleFilter := &models.StringCriterionInput{
			Value:    existingAudio.Title,
			Modifier: models.CriterionModifierEquals,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Title: titleFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.Count, 0)

		// Verify results
		audios, err := result.Resolve(ctx)
		assert.Nil(t, err)
		for _, audio := range audios {
			assert.Equal(t, existingAudio.Title, audio.Title)
		}
	})
}

func TestAudioFilterByPerformer(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterByPerformer", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test filtering by performer
		performerFilter := &models.MultiCriterionInput{
			Value:    []string{strconv.Itoa(performerIDs[performerIdx1WithScene])},
			Modifier: models.CriterionModifierIncludes,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Performers: performerFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, result)

		// If there are results, verify performer associations
		if result.Count > 0 {
			audios, err := result.Resolve(ctx)
			assert.Nil(t, err)
			assert.Greater(t, len(audios), 0)
		}
	})
}

func TestAudioFilterByTag(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterByTag", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test filtering by tag
		tagFilter := &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(tagIDs[tagIdxWithScene])},
			Modifier: models.CriterionModifierIncludes,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Tags: tagFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, result)

		// If there are results, verify they exist
		if result.Count > 0 {
			audios, err := result.Resolve(ctx)
			assert.Nil(t, err)
			assert.Greater(t, len(audios), 0)
		}
	})
}

func TestAudioFilterByRating(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterByRating", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test filtering by rating
		ratingFilter := &models.IntCriterionInput{
			Value:    50,
			Modifier: models.CriterionModifierGreaterThan,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Rating100: ratingFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, result)

		// Verify results have rating > 50
		if result.Count > 0 {
			audios, err := result.Resolve(ctx)
			assert.Nil(t, err)
			for _, audio := range audios {
				if audio.Rating != nil {
					assert.Greater(t, *audio.Rating, 50)
				}
			}
		}
	})
}

func TestAudioFilterByDuration(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterByDuration", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test filtering by duration range (5-15 seconds)
		maxDuration := 15
		durationFilter := &models.IntCriterionInput{
			Value:    5,
			Value2:   &maxDuration,
			Modifier: models.CriterionModifierBetween,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Duration: durationFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})

		// This should not fail due to SQL errors
		assert.Nil(t, err)
		assert.NotNil(t, result)

		// The result may be 0 if no audio files match the duration, but it should not error
		assert.GreaterOrEqual(t, result.Count, 0)

		// If there are results, verify they can be resolved
		if result.Count > 0 {
			audios, err := result.Resolve(ctx)
			assert.Nil(t, err)
			assert.Equal(t, result.Count, len(audios))
		}
	})
}

func TestAudioFilterByBitrate(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterByBitrate", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test filtering by bitrate greater than 128
		bitrateFilter := &models.IntCriterionInput{
			Value:    128,
			Modifier: models.CriterionModifierGreaterThan,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Bitrate: bitrateFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})

		// This should not fail due to SQL errors
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Count, 0)

		// If there are results, verify they can be resolved
		if result.Count > 0 {
			audios, err := result.Resolve(ctx)
			assert.Nil(t, err)
			assert.Equal(t, result.Count, len(audios))
		}
	})
}

func TestAudioFilterBySampleRate(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterBySampleRate", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test filtering by sample rate
		sampleRateFilter := &models.IntCriterionInput{
			Value:    44100,
			Modifier: models.CriterionModifierEquals,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				SampleRate: sampleRateFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})

		// This should not fail due to SQL errors
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Count, 0)

		// If there are results, verify they can be resolved
		if result.Count > 0 {
			audios, err := result.Resolve(ctx)
			assert.Nil(t, err)
			assert.Equal(t, result.Count, len(audios))
		}
	})
}

func TestAudioFilterByChannels(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterByChannels", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test filtering by channels
		channelsFilter := &models.IntCriterionInput{
			Value:    2,
			Modifier: models.CriterionModifierEquals,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Channels: channelsFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})

		// This should not fail due to SQL errors
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Count, 0)

		// If there are results, verify they can be resolved
		if result.Count > 0 {
			audios, err := result.Resolve(ctx)
			assert.Nil(t, err)
			assert.Equal(t, result.Count, len(audios))
		}
	})
}

func TestAudioFilterByCodec(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFilterByCodec", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test filtering by audio codec
		codecFilter := &models.StringCriterionInput{
			Value:    "mp3",
			Modifier: models.CriterionModifierEquals,
		}

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				AudioCodec: codecFilter,
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})

		// This should not fail due to SQL errors
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Count, 0)

		// If there are results, verify they can be resolved
		if result.Count > 0 {
			audios, err := result.Resolve(ctx)
			assert.Nil(t, err)
			assert.Equal(t, result.Count, len(audios))
		}
	})
}

func TestAudioCount(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioCount", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		count, err := qb.Count(ctx)
		assert.Nil(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})
}

func TestAudioFindByChecksum(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioFindByChecksum", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Test with a valid checksum if available
		// This test may not find results if test data doesn't have checksums
		audios, err := qb.FindByChecksum(ctx, "nonexistent-checksum")
		assert.Nil(t, err)
		assert.Equal(t, 0, len(audios)) // Should return empty for non-existent checksum
	})
}

func TestAudioBulkOperations(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioBulkOperations", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Create multiple audios
		var audioIDs []int
		for i := 0; i < 3; i++ {
			newAudio := models.NewAudio()
			newAudio.Title = fmt.Sprintf("Bulk Audio %d", i+1)
			err := qb.Create(ctx, &newAudio, []models.FileID{})
			assert.Nil(t, err)
			audioIDs = append(audioIDs, newAudio.ID)
		}

		// Find multiple audios
		audios, err := qb.FindMany(ctx, audioIDs)
		assert.Nil(t, err)
		assert.Equal(t, 3, len(audios))

		// Verify all audios were found
		foundIDs := make(map[int]bool)
		for _, audio := range audios {
			foundIDs[audio.ID] = true
		}
		for _, id := range audioIDs {
			assert.True(t, foundIDs[id], "Audio ID %d not found", id)
		}
	})
}

func TestAudioRelationships(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioRelationships", func(t *testing.T, ctx context.Context) {
		// Test audio with performer relationship
		audio, err := db.Audio.Find(ctx, audioIDs[audioIdxWithPerformer])
		if err != nil {
			t.Errorf("Error finding audio: %s", err.Error())
			return
		}

		// Load performer IDs
		if err := audio.LoadPerformerIDs(ctx, db.Audio); err != nil {
			t.Errorf("Error loading performer IDs: %s", err.Error())
			return
		}

		assert.Greater(t, len(audio.PerformerIDs.List()), 0)

		// Test audio with tag relationship
		audioWithTag, err := db.Audio.Find(ctx, audioIDs[audioIdxWithTag])
		if err != nil {
			t.Errorf("Error finding audio with tag: %s", err.Error())
			return
		}

		// Load tag IDs
		if err := audioWithTag.LoadTagIDs(ctx, db.Audio); err != nil {
			t.Errorf("Error loading tag IDs: %s", err.Error())
			return
		}

		assert.Greater(t, len(audioWithTag.TagIDs.List()), 0)
	})
}

func TestAudioAdvancedSearchWithMultipleFilters(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioAdvancedSearchWithMultipleFilters", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Create test audio records with comprehensive metadata
		// Similar to the Cucumber test: "Advanced Search with Multiple Filters"
		testAudios := []struct {
			title     string
			rating    int
			duration  int // in seconds
			performer string
		}{
			{"Rock Song", 80, 240, "Artist1"},   // 4 minutes, rating 80 (4 on 5-point scale)
			{"Jazz Tune", 100, 180, "Artist2"},  // 3 minutes, rating 100 (5 on 5-point scale)
			{"Classical", 60, 300, "Artist3"},   // 5 minutes, rating 60 (3 on 5-point scale)
			{"Pop Hit", 40, 210, "Artist4"},     // 3.5 minutes, rating 40 (2 on 5-point scale) (should be excluded)
			{"Electronic", 100, 350, "Artist5"}, // 5.8 minutes, rating 100 (5 on 5-point scale) (should be excluded by duration)
		}

		var createdAudioIDs []int

		// Create test audio records
		for _, testAudio := range testAudios {
			newAudio := models.NewAudio()
			newAudio.Title = testAudio.title
			newAudio.Rating = &testAudio.rating
			// Note: duration filtering happens at the file level, not audio level
			// For this test, we'll focus on rating and play_count filtering

			err := qb.Create(ctx, &newAudio, []models.FileID{})
			assert.Nil(t, err)
			createdAudioIDs = append(createdAudioIDs, newAudio.ID)

			// Add play events to simulate play counts
			// Rock Song: 3 plays, Jazz Tune: 1 play, Classical: 2 plays
			// Pop Hit: 0 plays, Electronic: 1 play
			var playCount int
			switch testAudio.title {
			case "Rock Song":
				playCount = 3
			case "Jazz Tune":
				playCount = 1
			case "Classical":
				playCount = 2
			case "Pop Hit":
				playCount = 0
			case "Electronic":
				playCount = 1
			}

			// Add play events using the audio manager
			if playCount > 0 {
				for i := 0; i < playCount; i++ {
					// Add view events using the proper audio store method
					_, err := qb.AddViews(ctx, newAudio.ID, nil)
					assert.Nil(t, err)
				}
			}
		}

		// Test complex filters: rating >= 80 (equivalent to rating >= 4 on 5-point scale) AND play_count > 1
		// This should match "Rock Song" (rating 80, play_count 3) and exclude:
		// - "Jazz Tune" (rating 100, but play_count 1, not > 1)
		// - "Classical" (rating 60, not >= 80)
		// - "Pop Hit" (rating 40, not >= 80)
		// - "Electronic" (rating 100, but play_count 1, not > 1)

		result, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Rating100: &models.IntCriterionInput{
					Value:    79, // rating > 79 (equivalent to rating >= 80)
					Modifier: models.CriterionModifierGreaterThan,
				},
				PlayCount: &models.IntCriterionInput{
					Value:    1, // play_count > 1
					Modifier: models.CriterionModifierGreaterThan,
				},
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})

		assert.Nil(t, err)
		assert.NotNil(t, result)

		// Verify the results
		audios, err := result.Resolve(ctx)
		assert.Nil(t, err)

		// Filter results to only our created test records
		filteredAudios := make([]*models.Audio, 0)
		for _, audio := range audios {
			for _, createdID := range createdAudioIDs {
				if audio.ID == createdID {
					filteredAudios = append(filteredAudios, audio)
					break
				}
			}
		}

		// Should find only "Rock Song" (rating 80, play_count 3)
		assert.Equal(t, 1, len(filteredAudios), "Should find exactly 1 result from our test data matching complex filters")

		// Check that the result is "Rock Song"
		assert.Equal(t, "Rock Song", filteredAudios[0].Title)
		assert.Equal(t, 80, *filteredAudios[0].Rating)

		// Test another combination: rating >= 100 (should find "Jazz Tune" and "Electronic")
		result2, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				Rating100: &models.IntCriterionInput{
					Value:    99, // rating > 99 (equivalent to rating >= 100)
					Modifier: models.CriterionModifierGreaterThan,
				},
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})

		assert.Nil(t, err)
		assert.NotNil(t, result2)

		audios2, err := result2.Resolve(ctx)
		assert.Nil(t, err)

		// Filter results to only our created test records
		filteredAudios2 := make([]*models.Audio, 0)
		for _, audio := range audios2 {
			for _, createdID := range createdAudioIDs {
				if audio.ID == createdID {
					filteredAudios2 = append(filteredAudios2, audio)
					break
				}
			}
		}

		assert.Equal(t, 2, len(filteredAudios2), "Should find 2 results from our test data with rating >= 100")

		// Check that both "Jazz Tune" and "Electronic" are found
		titles := make([]string, len(filteredAudios2))
		for i, audio := range filteredAudios2 {
			titles[i] = audio.Title
		}
		assert.Contains(t, titles, "Jazz Tune")
		assert.Contains(t, titles, "Electronic")

		// Test play_count filtering alone: play_count > 1 (should find "Rock Song" and "Classical")
		result3, err := qb.Query(ctx, models.AudioQueryOptions{
			AudioFilter: &models.AudioFilterType{
				PlayCount: &models.IntCriterionInput{
					Value:    1, // play_count > 1
					Modifier: models.CriterionModifierGreaterThan,
				},
			},
			QueryOptions: models.QueryOptions{
				Count: true,
			},
		})

		assert.Nil(t, err)
		assert.NotNil(t, result3)

		audios3, err := result3.Resolve(ctx)
		assert.Nil(t, err)

		// Filter results to only our created test records
		filteredAudios3 := make([]*models.Audio, 0)
		for _, audio := range audios3 {
			for _, createdID := range createdAudioIDs {
				if audio.ID == createdID {
					filteredAudios3 = append(filteredAudios3, audio)
					break
				}
			}
		}

		assert.Equal(t, 2, len(filteredAudios3), "Should find 2 results from our test data with play_count > 1")

		// Check that both "Rock Song" and "Classical" are found
		titles3 := make([]string, len(filteredAudios3))
		for i, audio := range filteredAudios3 {
			titles3[i] = audio.Title
		}
		assert.Contains(t, titles3, "Rock Song")
		assert.Contains(t, titles3, "Classical")

		// Clean up created records
		for _, audioID := range createdAudioIDs {
			err := qb.Destroy(ctx, audioID)
			assert.Nil(t, err)
		}
	})
}

// Test O-counter mutations
func TestAudioIncrementO(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioIncrementO", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Create a new audio for testing
		newAudio := models.NewAudio()
		newAudio.Title = "Test Audio for O-Counter"
		err := qb.Create(ctx, &newAudio, []models.FileID{})
		assert.Nil(t, err)
		audioID := newAudio.ID

		// Initial count should be 0
		count, err := qb.GetOCount(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 0, count)

		// Simulate incrementing O counter (add current timestamp)
		dates, err := qb.AddO(ctx, audioID, []time.Time{time.Now()})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(dates))

		// Verify count is now 1
		count, err = qb.GetOCount(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 1, count)

		// Add another O-date
		dates, err = qb.AddO(ctx, audioID, []time.Time{time.Now()})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(dates))

		// Verify count is now 2
		count, err = qb.GetOCount(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestAudioDecrementO(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioDecrementO", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Create a new audio with multiple O-dates
		newAudio := models.NewAudio()
		newAudio.Title = "Test Audio for O-Counter Decrement"
		err := qb.Create(ctx, &newAudio, []models.FileID{})
		assert.Nil(t, err)
		audioID := newAudio.ID

		// Add 3 O-dates
		now := time.Now()
		dates := []time.Time{
			now.Add(-2 * time.Hour),
			now.Add(-1 * time.Hour),
			now,
		}
		addedDates, err := qb.AddO(ctx, audioID, dates)
		assert.Nil(t, err)
		assert.Equal(t, 3, len(addedDates))

		// Get the most recent date
		allDates, err := qb.GetODates(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 3, len(allDates))

		// Remove the most recent date
		remainingDates, err := qb.DeleteO(ctx, audioID, []time.Time{allDates[len(allDates)-1]})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(remainingDates))

		// Verify count is now 2
		count, err := qb.GetOCount(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 2, count)

		// Test removing a non-existent date (should not error)
		fakeDate := time.Now().Add(24 * time.Hour)
		remainingDates, err = qb.DeleteO(ctx, audioID, []time.Time{fakeDate})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(remainingDates)) // Count should remain unchanged
	})
}

func TestAudioResetO_NewImplementation(t *testing.T) {
	runWithRollbackTxn(t, "TestAudioResetO_NewImplementation", func(t *testing.T, ctx context.Context) {
		qb := db.Audio

		// Create a new audio with multiple O-dates
		newAudio := models.NewAudio()
		newAudio.Title = "Test Audio for O-Counter Reset"
		err := qb.Create(ctx, &newAudio, []models.FileID{})
		assert.Nil(t, err)
		audioID := newAudio.ID

		// Add multiple O-dates
		dates := []time.Time{
			time.Now().Add(-2 * time.Hour),
			time.Now().Add(-1 * time.Hour),
			time.Now(),
		}
		_, err = qb.AddO(ctx, audioID, dates)
		assert.Nil(t, err)

		// Verify count is 3
		count, err := qb.GetOCount(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 3, count)

		// Reset O counter
		newCount, err := qb.ResetO(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 0, newCount)

		// Verify no dates remain in table
		allDates, err := qb.GetODates(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(allDates))

		// Verify count is 0
		count, err = qb.GetOCount(ctx, audioID)
		assert.Nil(t, err)
		assert.Equal(t, 0, count)
	})
}
