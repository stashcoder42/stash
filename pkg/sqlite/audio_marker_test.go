//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ = stringslice.StringSliceToIntSlice // keep import for potential future use

func audioMarkersToIDs(i []*models.AudioMarker) []int {
	ret := make([]int, len(i))
	for i, v := range i {
		ret[i] = v.ID
	}

	return ret
}

func queryAudioMarkers(ctx context.Context, t *testing.T, sqb models.AudioMarkerReader, markerFilter *models.AudioMarkerFilterType, findFilter *models.FindFilterType) []*models.AudioMarker {
	t.Helper()
	result, _, err := sqb.Query(ctx, markerFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying audio markers: %v", err)
	}

	return result
}

func TestAudioMarkerFindByAudioID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		mqb := db.AudioMarker

		audioID := audioIDs[audioIdxWithMarkers]
		markers, err := mqb.FindByAudioID(ctx, audioID)

		if err != nil {
			t.Errorf("Error finding markers: %s", err.Error())
		}

		assert.Greater(t, len(markers), 0)
		for _, marker := range markers {
			assert.Equal(t, audioIDs[audioIdxWithMarkers], marker.AudioID)
		}

		markers, err = mqb.FindByAudioID(ctx, 0)

		if err != nil {
			t.Errorf("Error finding marker: %s", err.Error())
		}

		assert.Len(t, markers, 0)

		return nil
	})
}

func TestAudioMarkerCountByTagID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		mqb := db.AudioMarker

		markerCount, err := mqb.CountByTagID(ctx, tagIDs[tagIdxWithPrimaryMarkers])

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		// 5 audio markers use tagIdxWithPrimaryMarkers as primary tag
		assert.Equal(t, 5, markerCount)

		markerCount, err = mqb.CountByTagID(ctx, tagIDs[tagIdxWithMarkers])

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		// 2 audio markers have tagIdxWithMarkers as a tag
		assert.Equal(t, 2, markerCount)

		markerCount, err = mqb.CountByTagID(ctx, 0)

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		assert.Equal(t, 0, markerCount)

		return nil
	})
}

func TestAudioMarkerQueryQ(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		q := getAudioStringValue(audioIdxWithMarkers, titleField)
		m, _, err := db.AudioMarker.Query(ctx, nil, &models.FindFilterType{
			Q: &q,
		})

		if err != nil {
			t.Errorf("Error querying audio markers: %s", err.Error())
		}

		if !assert.Greater(t, len(m), 0) {
			return nil
		}

		assert.Equal(t, audioIDs[audioIdxWithMarkers], m[0].AudioID)

		return nil
	})
}

func TestAudioMarkerQuerySortByAudioUpdated(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sort := "audios_updated_at"
		_, _, err := db.AudioMarker.Query(ctx, nil, &models.FindFilterType{
			Sort: &sort,
		})

		if err != nil {
			t.Errorf("Error querying audio markers: %s", err.Error())
		}

		return nil
	})
}

func TestAudioMarkerQueryTags(t *testing.T) {
	type test struct {
		name         string
		markerFilter *models.AudioMarkerFilterType
		findFilter   *models.FindFilterType
	}

	withTxn(func(ctx context.Context) error {
		testTags := func(t *testing.T, m *models.AudioMarker, markerFilter *models.AudioMarkerFilterType) {
			tagIDs, err := db.AudioMarker.GetTagIDs(ctx, m.ID)
			if err != nil {
				t.Errorf("error getting marker tag ids: %v", err)
			}

			// HACK - if modifier isn't null/not null, then add the primary tag id
			if markerFilter.Tags.Modifier != models.CriterionModifierIsNull && markerFilter.Tags.Modifier != models.CriterionModifierNotNull {
				tagIDs = append(tagIDs, m.PrimaryTagID)
			}

			values, _ := stringslice.StringSliceToIntSlice(markerFilter.Tags.Value)
			verifyIDs(t, markerFilter.Tags.Modifier, values, tagIDs)
		}

		cases := []test{
			{
				"is null",
				&models.AudioMarkerFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIsNull,
					},
				},
				nil,
			},
			{
				"not null",
				&models.AudioMarkerFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierNotNull,
					},
				},
				nil,
			},
			{
				"includes",
				&models.AudioMarkerFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludes,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdxWithMarkers]),
						},
					},
				},
				nil,
			},
			{
				"includes all",
				&models.AudioMarkerFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludesAll,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdxWithMarkers]),
							strconv.Itoa(tagIDs[tagIdx2WithMarkers]),
						},
					},
				},
				nil,
			},
			{
				"equals",
				&models.AudioMarkerFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierEquals,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdxWithPrimaryMarkers]),
							strconv.Itoa(tagIDs[tagIdxWithMarkers]),
							strconv.Itoa(tagIDs[tagIdx2WithMarkers]),
						},
					},
				},
				nil,
			},
			{
				"excludes",
				&models.AudioMarkerFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludes,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdx2WithMarkers]),
						},
					},
				},
				nil,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				markers := queryAudioMarkers(ctx, t, db.AudioMarker, tc.markerFilter, tc.findFilter)
				assert.Greater(t, len(markers), 0)
				for _, m := range markers {
					testTags(t, m, tc.markerFilter)
				}
			})
		}

		return nil
	})
}

// Note: TestAudioMarkerQueryAudioTags and TestAudioMarkerQueryDuration tests
// are not implemented because AudioMarkerFilterType doesn't have AudioTags or Duration fields.
// These fields exist in SceneMarkerFilterType but weren't added to AudioMarkerFilterType.
// TODO: Add AudioTags and Duration fields to AudioMarkerFilterType if needed.

// CRUD Tests (ported from audio/scraping branch)

func TestAudioMarkerStoreBasicOperations(t *testing.T) {
	runWithRollbackTxn(t, "basic operations", func(t *testing.T, ctx context.Context) {
		assert := assert.New(t)
		require := require.New(t)

		qb := db.AudioMarker

		// Test Count - should have audioMarkerSpecs markers
		count, err := qb.Count(ctx)
		require.NoError(err)
		assert.Equal(len(audioMarkerSpecs), count)

		// Test All - should return all markers
		all, err := qb.All(ctx)
		require.NoError(err)
		assert.Len(all, len(audioMarkerSpecs))
	})
}

func TestAudioMarkerStoreCreate(t *testing.T) {
	runWithRollbackTxn(t, "create", func(t *testing.T, ctx context.Context) {
		assert := assert.New(t)
		require := require.New(t)

		qb := db.AudioMarker

		// Create an audio marker using existing audio and tag fixtures
		marker := models.NewAudioMarker()
		marker.Title = "Test Audio Marker"
		marker.Seconds = 30.5
		marker.PrimaryTagID = tagIDs[tagIdxWithScene]
		marker.AudioID = audioIDs[audioIdxWithPerformer]

		// Test Create
		err := qb.Create(ctx, &marker)
		require.NoError(err)
		assert.NotZero(marker.ID)
		assert.Equal("Test Audio Marker", marker.Title)
		assert.Equal(30.5, marker.Seconds)
		assert.Equal(tagIDs[tagIdxWithScene], marker.PrimaryTagID)
		assert.Equal(audioIDs[audioIdxWithPerformer], marker.AudioID)

		// Test Find
		found, err := qb.Find(ctx, marker.ID)
		require.NoError(err)
		require.NotNil(found)
		assert.Equal(marker.ID, found.ID)
		assert.Equal(marker.Title, found.Title)
		assert.Equal(marker.Seconds, found.Seconds)

		// Test FindMany
		many, err := qb.FindMany(ctx, []int{marker.ID})
		require.NoError(err)
		require.Len(many, 1)
		assert.Equal(marker.ID, many[0].ID)

		// Test FindByAudioID
		audioMarkers, err := qb.FindByAudioID(ctx, audioIDs[audioIdxWithPerformer])
		require.NoError(err)
		require.Len(audioMarkers, 1)
		assert.Equal(marker.ID, audioMarkers[0].ID)
	})
}

func TestAudioMarkerStoreUpdatePartial(t *testing.T) {
	runWithRollbackTxn(t, "update partial", func(t *testing.T, ctx context.Context) {
		assert := assert.New(t)
		require := require.New(t)

		qb := db.AudioMarker

		// Create audio marker
		marker := models.NewAudioMarker()
		marker.Title = "Original Title"
		marker.Seconds = 10.0
		marker.PrimaryTagID = tagIDs[tagIdxWithScene]
		marker.AudioID = audioIDs[audioIdxWithPerformer]

		err := qb.Create(ctx, &marker)
		require.NoError(err)

		// Test UpdatePartial
		partial := models.NewAudioMarkerPartial()
		newTitle := "Updated Title"
		newSeconds := 25.5
		partial.Title = models.NewOptionalString(newTitle)
		partial.Seconds = models.NewOptionalFloat64(newSeconds)

		updated, err := qb.UpdatePartial(ctx, marker.ID, partial)
		require.NoError(err)
		require.NotNil(updated)

		assert.Equal(newTitle, updated.Title)
		assert.Equal(newSeconds, updated.Seconds)
		assert.Equal(marker.PrimaryTagID, updated.PrimaryTagID) // Unchanged
		assert.Equal(marker.AudioID, updated.AudioID)           // Unchanged
	})
}

func TestAudioMarkerStoreDestroy(t *testing.T) {
	runWithRollbackTxn(t, "destroy", func(t *testing.T, ctx context.Context) {
		assert := assert.New(t)
		require := require.New(t)

		qb := db.AudioMarker

		// Create audio marker
		marker := models.NewAudioMarker()
		marker.Title = "To Be Destroyed"
		marker.Seconds = 15.0
		marker.PrimaryTagID = tagIDs[tagIdxWithScene]
		marker.AudioID = audioIDs[audioIdxWithPerformer]

		err := qb.Create(ctx, &marker)
		require.NoError(err)

		// Verify it exists
		found, err := qb.Find(ctx, marker.ID)
		require.NoError(err)
		require.NotNil(found)

		// Test Destroy
		err = qb.Destroy(ctx, marker.ID)
		require.NoError(err)

		// Verify it's gone
		found, err = qb.Find(ctx, marker.ID)
		require.NoError(err)
		assert.Nil(found)
	})
}

func TestAudioMarkerStoreTags(t *testing.T) {
	runWithRollbackTxn(t, "tags", func(t *testing.T, ctx context.Context) {
		assert := assert.New(t)
		require := require.New(t)

		qb := db.AudioMarker

		// Create audio marker
		marker := models.NewAudioMarker()
		marker.Title = "Marker With Tags"
		marker.Seconds = 20.0
		marker.PrimaryTagID = tagIDs[tagIdxWithScene]
		marker.AudioID = audioIDs[audioIdxWithPerformer]

		err := qb.Create(ctx, &marker)
		require.NoError(err)

		// Test GetTagIDs - should be empty initially
		tagIDsResult, err := qb.GetTagIDs(ctx, marker.ID)
		require.NoError(err)
		assert.Empty(tagIDsResult)

		// Test UpdateTags - add some tags
		newTagIDs := []int{tagIDs[tagIdxWithMarkers], tagIDs[tagIdx2WithMarkers]}
		err = qb.UpdateTags(ctx, marker.ID, newTagIDs)
		require.NoError(err)

		// Verify tags were added
		tagIDsResult, err = qb.GetTagIDs(ctx, marker.ID)
		require.NoError(err)
		assert.Len(tagIDsResult, 2)
		assert.Contains(tagIDsResult, tagIDs[tagIdxWithMarkers])
		assert.Contains(tagIDsResult, tagIDs[tagIdx2WithMarkers])

		// Test UpdateTags - replace with different tags
		replaceTags := []int{tagIDs[tagIdx1WithScene]}
		err = qb.UpdateTags(ctx, marker.ID, replaceTags)
		require.NoError(err)

		tagIDsResult, err = qb.GetTagIDs(ctx, marker.ID)
		require.NoError(err)
		assert.Len(tagIDsResult, 1)
		assert.Contains(tagIDsResult, tagIDs[tagIdx1WithScene])

		// Test UpdateTags - clear all tags
		err = qb.UpdateTags(ctx, marker.ID, []int{})
		require.NoError(err)

		tagIDsResult, err = qb.GetTagIDs(ctx, marker.ID)
		require.NoError(err)
		assert.Empty(tagIDsResult)
	})
}
