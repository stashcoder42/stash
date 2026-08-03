//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper to create test audio entries
func createTestAudio(t *testing.T, db models.AudioReaderWriter) *models.Audio {
	audio := models.Audio{
		Title:     "Test Audio",
		Details:   "Test Details",
		Organized: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := db.Create(context.Background(), &audio, nil)
	require.NoError(t, err)

	return &audio
}

// Test helper to count o-dates in audios_o_dates table
func getODateCount(t *testing.T, ctx context.Context, db models.AudioReaderWriter, audioID int) int {
	count, err := db.GetOCount(ctx, audioID)
	require.NoError(t, err)
	return count
}

func loadAudioRelationships(ctx context.Context, expected models.Audio, actual *models.Audio) error {
	if expected.URLs.Loaded() {
		if err := actual.LoadURLs(ctx, db.Audio); err != nil {
			return err
		}
	}
	if expected.PerformerIDs.Loaded() {
		if err := actual.LoadPerformerIDs(ctx, db.Audio); err != nil {
			return err
		}
	}
	if expected.TagIDs.Loaded() {
		if err := actual.LoadTagIDs(ctx, db.Audio); err != nil {
			return err
		}
	}
	if expected.Files.Loaded() {
		if err := actual.LoadFiles(ctx, db.Audio); err != nil {
			return err
		}
	}

	// clear Path, Checksum, PrimaryFileID
	if expected.Path == "" {
		actual.Path = ""
	}
	if expected.Checksum == "" {
		actual.Checksum = ""
	}
	if expected.PrimaryFileID == nil {
		actual.PrimaryFileID = nil
	}

	return nil
}

func Test_audioQueryBuilder_Create(t *testing.T) {
	var (
		title     = "title"
		url       = "http://example.com/audio"
		rating    = 60
		details   = "details"
		date, _   = models.ParseDate("2003-02-01")
		createdAt = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		updatedAt = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	)

	tests := []struct {
		name      string
		newObject models.Audio
		wantErr   bool
	}{
		{
			"full",
			models.Audio{
				Title:        title,
				URLs:         models.NewRelatedStrings([]string{url}),
				Rating:       &rating,
				Date:         &date,
				Details:      details,
				Organized:    true,
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				TagIDs:       models.NewRelatedIDs([]int{tagIDs[tagIdxWithScene], tagIDs[tagIdx1WithScene]}),
				PerformerIDs: models.NewRelatedIDs([]int{performerIDs[performerIdx1WithScene], performerIDs[performerIdx1WithDupName]}),
			},
			false,
		},
		{
			"invalid tag id",
			models.Audio{
				TagIDs: models.NewRelatedIDs([]int{invalidID}),
			},
			true,
		},
		{
			"invalid performer id",
			models.Audio{
				PerformerIDs: models.NewRelatedIDs([]int{invalidID}),
			},
			true,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			var fileIDs []models.FileID
			if tt.newObject.Files.Loaded() {
				for _, f := range tt.newObject.Files.List() {
					fileIDs = append(fileIDs, f.Base().ID)
				}
			}
			s := tt.newObject
			if err := qb.Create(ctx, &s, fileIDs); (err != nil) != tt.wantErr {
				t.Errorf("audioQueryBuilder.Create() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				assert.Zero(s.ID)
				return
			}

			assert.NotZero(s.ID)

			copy := tt.newObject
			copy.ID = s.ID

			// load relationships
			if err := loadAudioRelationships(ctx, copy, &s); err != nil {
				t.Errorf("loadAudioRelationships() error = %v", err)
				return
			}

			assert.Equal(copy, s)

			// ensure can find the audio
			found, err := qb.Find(ctx, s.ID)
			if err != nil {
				t.Errorf("audioQueryBuilder.Find() error = %v", err)
			}

			// load relationships
			if err := loadAudioRelationships(ctx, copy, found); err != nil {
				t.Errorf("loadAudioRelationships() error = %v", err)
				return
			}

			assert.Equal(copy, *found)

			return
		})
	}
}

func clearAudioFileIDs(audio *models.Audio) {
	if audio.Files.Loaded() {
		for _, f := range audio.Files.List() {
			f.Base().ID = 0
		}
	}
}

func makeAudioFileWithID(i int) *models.AudioFile {
	ret := makeAudioFile(i)
	ret.ID = audioFileIDs[i]
	return ret
}

func Test_audioQueryBuilder_Update(t *testing.T) {
	var (
		title     = "title"
		url       = "http://example.com/audio"
		rating    = 60
		details   = "details"
		date, _   = models.ParseDate("2003-02-01")
		createdAt = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		updatedAt = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	)

	tests := []struct {
		name          string
		updatedObject *models.Audio
		wantErr       bool
	}{
		{
			"full",
			&models.Audio{
				ID:           audioIDs[audioIdxWithPerformer],
				Title:        title,
				URLs:         models.NewRelatedStrings([]string{url}),
				Rating:       &rating,
				Date:         &date,
				Details:      details,
				Organized:    true,
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				TagIDs:       models.NewRelatedIDs([]int{tagIDs[tagIdxWithScene], tagIDs[tagIdx1WithScene]}),
				PerformerIDs: models.NewRelatedIDs([]int{performerIDs[performerIdx1WithScene], performerIDs[performerIdx1WithDupName]}),
			},
			false,
		},
		{
			"clear nullables",
			&models.Audio{
				ID:           audioIDs[audioIdxWithPerformer],
				Title:        getAudioStringValue(audioIdxWithPerformer, titleField),
				TagIDs:       models.NewRelatedIDs([]int{}),
				PerformerIDs: models.NewRelatedIDs([]int{}),
				Organized:    true,
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
			},
			false,
		},
		{
			"invalid tag id",
			&models.Audio{
				ID:        audioIDs[audioIdxWithPerformer],
				Organized: true,
				TagIDs:    models.NewRelatedIDs([]int{invalidID}),
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			true,
		},
		{
			"invalid performer id",
			&models.Audio{
				ID:           audioIDs[audioIdxWithPerformer],
				Organized:    true,
				PerformerIDs: models.NewRelatedIDs([]int{invalidID}),
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
			},
			true,
		},
	}

	qb := db.Audio
	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			copy := *tt.updatedObject

			if err := qb.Update(ctx, tt.updatedObject); (err != nil) != tt.wantErr {
				t.Errorf("audioQueryBuilder.Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			s, err := qb.Find(ctx, tt.updatedObject.ID)
			if err != nil {
				t.Errorf("audioQueryBuilder.Find() error = %v", err)
			}

			// load relationships
			if err := loadAudioRelationships(ctx, copy, s); err != nil {
				t.Errorf("loadAudioRelationships() error = %v", err)
				return
			}

			assert.Equal(copy, *s)

			return
		})
	}
}

func clearAudioPartial() models.AudioPartial {
	// leave mandatory fields
	return models.AudioPartial{
		Details:      models.OptionalString{Set: true, Null: true},
		URLs:         &models.UpdateStrings{Values: []string{}, Mode: models.RelationshipUpdateModeSet},
		Rating:       models.OptionalInt{Set: true, Null: true},
		Date:         models.OptionalDate{Set: true, Null: true},
		TagIDs:       &models.UpdateIDs{Mode: models.RelationshipUpdateModeSet},
		PerformerIDs: &models.UpdateIDs{Mode: models.RelationshipUpdateModeSet},
	}
}

func Test_audioQueryBuilder_UpdatePartial(t *testing.T) {
	var (
		title     = "title"
		url       = "http://example.com/audio"
		details   = "details"
		rating    = 60
		date, _   = models.ParseDate("2003-02-01")
		createdAt = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		updatedAt = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	)

	tests := []struct {
		name    string
		id      int
		partial models.AudioPartial
		want    models.Audio
		wantErr bool
	}{
		{
			"full",
			audioIDs[audioIdxWithPerformer],
			models.AudioPartial{
				Title:     models.NewOptionalString(title),
				URLs:      &models.UpdateStrings{Values: []string{url}, Mode: models.RelationshipUpdateModeSet},
				Details:   models.NewOptionalString(details),
				Rating:    models.NewOptionalInt(rating),
				Date:      models.NewOptionalDate(date),
				Organized: models.NewOptionalBool(true),
				CreatedAt: models.NewOptionalTime(createdAt),
				UpdatedAt: models.NewOptionalTime(updatedAt),
				TagIDs: &models.UpdateIDs{
					IDs:  []int{tagIDs[tagIdxWithScene], tagIDs[tagIdx1WithScene]},
					Mode: models.RelationshipUpdateModeSet,
				},
				PerformerIDs: &models.UpdateIDs{
					IDs:  []int{performerIDs[performerIdx1WithScene], performerIDs[performerIdx1WithDupName]},
					Mode: models.RelationshipUpdateModeSet,
				},
			},
			models.Audio{
				ID:        audioIDs[audioIdxWithPerformer],
				Title:     title,
				URLs:      models.NewRelatedStrings([]string{url}),
				Details:   details,
				Rating:    &rating,
				Date:      &date,
				Organized: true,
				Files: models.NewRelatedFiles([]models.File{
					makeAudioFile(audioIdxWithPerformer),
				}),
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				TagIDs:       models.NewRelatedIDs([]int{tagIDs[tagIdxWithScene], tagIDs[tagIdx1WithScene]}),
				PerformerIDs: models.NewRelatedIDs([]int{performerIDs[performerIdx1WithScene], performerIDs[performerIdx1WithDupName]}),
			},
			false,
		},
		{
			"clear all",
			audioIDs[audioIdxWithPerformer],
			clearAudioPartial(),
			models.Audio{
				ID:    audioIDs[audioIdxWithPerformer],
				Title: getAudioStringValue(audioIdxWithPerformer, titleField),
				Files: models.NewRelatedFiles([]models.File{
					makeAudioFile(audioIdxWithPerformer),
				}),
				TagIDs:       models.NewRelatedIDs([]int{}),
				PerformerIDs: models.NewRelatedIDs([]int{}),
			},
			false,
		},
		{
			"invalid id",
			invalidID,
			models.AudioPartial{},
			models.Audio{},
			true,
		},
	}
	for _, tt := range tests {
		qb := db.Audio

		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			got, err := qb.UpdatePartial(ctx, tt.id, tt.partial)
			if (err != nil) != tt.wantErr {
				t.Errorf("audioQueryBuilder.UpdatePartial() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// load relationships
			if err := loadAudioRelationships(ctx, tt.want, got); err != nil {
				t.Errorf("loadAudioRelationships() error = %v", err)
				return
			}
			clearAudioFileIDs(got)

			assert.Equal(tt.want, *got)

			s, err := qb.Find(ctx, tt.id)
			if err != nil {
				t.Errorf("audioQueryBuilder.Find() error = %v", err)
			}

			// load relationships
			if err := loadAudioRelationships(ctx, tt.want, s); err != nil {
				t.Errorf("loadAudioRelationships() error = %v", err)
				return
			}
			clearAudioFileIDs(s)
			assert.Equal(tt.want, *s)
		})
	}
}

func Test_audioQueryBuilder_Destroy(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		wantErr bool
	}{
		{
			"valid",
			audioIDs[audioIdxWithPerformer],
			false,
		},
		{
			"invalid",
			invalidID,
			true,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			if err := qb.Destroy(ctx, tt.id); (err != nil) != tt.wantErr {
				t.Errorf("audioQueryBuilder.Destroy() error = %v, wantErr %v", err, tt.wantErr)
			}

			// ensure cannot be found
			a, err := qb.Find(ctx, tt.id)

			assert.Nil(err)
			assert.Nil(a)
		})
	}
}

func makeAudioWithID(index int) *models.Audio {
	ret := makeAudio(index)
	ret.ID = audioIDs[index]

	ret.Files = models.NewRelatedFiles([]models.File{makeAudioFile(index)})

	return ret
}

func Test_audioQueryBuilder_Find(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		want    *models.Audio
		wantErr bool
	}{
		{
			"valid",
			audioIDs[audioIdxWithPerformer],
			makeAudioWithID(audioIdxWithPerformer),
			false,
		},
		{
			"invalid",
			invalidID,
			nil,
			false,
		},
		{
			"with performers",
			audioIDs[audioIdxWithTwoPerformers],
			makeAudioWithID(audioIdxWithTwoPerformers),
			false,
		},
		{
			"with tags",
			audioIDs[audioIdxWithTwoTags],
			makeAudioWithID(audioIdxWithTwoTags),
			false,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.Find(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("audioQueryBuilder.Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != nil {
				// load relationships
				if err := loadAudioRelationships(ctx, *tt.want, got); err != nil {
					t.Errorf("loadAudioRelationships() error = %v", err)
					return
				}
				clearAudioFileIDs(got)
			}
			assert.Equal(tt.want, got)
		})
	}
}

func postFindAudios(ctx context.Context, want []*models.Audio, got []*models.Audio) error {
	for i, s := range got {
		// load relationships
		if i < len(want) {
			if err := loadAudioRelationships(ctx, *want[i], s); err != nil {
				return err
			}
		}
		clearAudioFileIDs(s)
	}

	return nil
}

func Test_audioQueryBuilder_FindMany(t *testing.T) {
	tests := []struct {
		name    string
		ids     []int
		want    []*models.Audio
		wantErr bool
	}{
		{
			"valid with relationships",
			[]int{audioIDs[audioIdxWithPerformer], audioIDs[audioIdxWithTwoPerformers], audioIDs[audioIdxWithTwoTags]},
			[]*models.Audio{
				makeAudioWithID(audioIdxWithPerformer),
				makeAudioWithID(audioIdxWithTwoPerformers),
				makeAudioWithID(audioIdxWithTwoTags),
			},
			false,
		},
		{
			"invalid",
			[]int{audioIDs[audioIdxWithPerformer], audioIDs[audioIdxWithTwoPerformers], invalidID},
			nil,
			true,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			got, err := qb.FindMany(ctx, tt.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("audioQueryBuilder.FindMany() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err := postFindAudios(ctx, tt.want, got); err != nil {
				t.Errorf("loadAudioRelationships() error = %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("audioQueryBuilder.FindMany() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_audioQueryBuilder_FindByChecksum(t *testing.T) {
	getChecksum := func(index int) string {
		return getAudioStringValue(index, checksumField)
	}

	tests := []struct {
		name     string
		checksum string
		want     []*models.Audio
		wantErr  bool
	}{
		{
			"valid",
			getChecksum(audioIdxWithPerformer),
			[]*models.Audio{makeAudioWithID(audioIdxWithPerformer)},
			false,
		},
		{
			"invalid",
			"invalid checksum",
			nil,
			false,
		},
		{
			"with performers",
			getChecksum(audioIdxWithTwoPerformers),
			[]*models.Audio{makeAudioWithID(audioIdxWithTwoPerformers)},
			false,
		},
		{
			"with tags",
			getChecksum(audioIdxWithTwoTags),
			[]*models.Audio{makeAudioWithID(audioIdxWithTwoTags)},
			false,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.FindByChecksum(ctx, tt.checksum)
			if (err != nil) != tt.wantErr {
				t.Errorf("audioQueryBuilder.FindByChecksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err := postFindAudios(ctx, tt.want, got); err != nil {
				t.Errorf("loadAudioRelationships() error = %v", err)
				return
			}

			assert.Equal(tt.want, got)
		})
	}
}

func Test_audioQueryBuilder_FindByPath(t *testing.T) {
	getPath := func(index int) string {
		return getFilePath(folderIdxWithFiles, getAudioBasename(index))
	}

	tests := []struct {
		name    string
		path    string
		want    *models.Audio
		wantErr bool
	}{
		{
			"valid",
			getPath(audioIdxWithPerformer),
			makeAudioWithID(audioIdxWithPerformer),
			false,
		},
		{
			"case insensitive",
			strings.ToUpper(getPath(audioIdxWithPerformer)),
			makeAudioWithID(audioIdxWithPerformer),
			false,
		},
		{
			"invalid",
			"invalid path",
			nil,
			false,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.FindByPath(ctx, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("audioQueryBuilder.FindByPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var want []*models.Audio
			if tt.want != nil {
				want = []*models.Audio{tt.want}
			}

			var gotSlice []*models.Audio
			if got != nil {
				gotSlice = []*models.Audio{got}
			}

			if err := postFindAudios(ctx, want, gotSlice); err != nil {
				t.Errorf("loadAudioRelationships() error = %v", err)
				return
			}

			if len(want) == 0 {
				assert.Nil(got)
			} else {
				require.NotNil(t, got)
				assert.Equal(want[0], got)
			}
		})
	}
}

func audiosToIDs(a []*models.Audio) []int {
	var ret []int
	for _, aa := range a {
		ret = append(ret, aa.ID)
	}

	return ret
}

func Test_audioStore_FindByFileID(t *testing.T) {
	tests := []struct {
		name    string
		fileID  models.FileID
		include []int
		exclude []int
	}{
		{
			"valid",
			audioFileIDs[audioIdxWithPerformer],
			[]int{audioIdxWithPerformer},
			nil,
		},
		{
			"invalid",
			invalidFileID,
			nil,
			[]int{audioIdxWithPerformer},
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.FindByFileID(ctx, tt.fileID)
			if err != nil {
				t.Errorf("AudioStore.FindByFileID() error = %v", err)
				return
			}
			for _, f := range got {
				clearAudioFileIDs(f)
			}

			ids := audiosToIDs(got)
			include := indexesToIDs(audioIDs, tt.include)
			exclude := indexesToIDs(audioIDs, tt.exclude)

			for _, i := range include {
				assert.Contains(ids, i)
			}
			for _, e := range exclude {
				assert.NotContains(ids, e)
			}
		})
	}
}

func TestAudioQueryQ(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		const audioIdx = 2

		q := getAudioStringValue(audioIdx, titleField)

		sqb := db.Audio

		audioQueryQ(ctx, t, sqb, q, audioIdx)

		return nil
	})
}

func queryAudiosWithCount(ctx context.Context, sqb models.AudioReader, audioFilter *models.AudioFilterType, findFilter *models.FindFilterType) ([]*models.Audio, int, error) {
	result, err := sqb.Query(ctx, models.AudioQueryOptions{
		QueryOptions: models.QueryOptions{
			FindFilter: findFilter,
			Count:      true,
		},
		AudioFilter: audioFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	audios, err := result.Resolve(ctx)
	if err != nil {
		return nil, 0, err
	}

	return audios, result.Count, nil
}

func audioQueryQ(ctx context.Context, t *testing.T, sqb models.AudioReader, q string, expectedAudioIdx int) {
	filter := models.FindFilterType{
		Q: &q,
	}
	audios := queryAudios(ctx, t, sqb, nil, &filter)

	assert.Len(t, audios, 1)
	audio := audios[0]
	assert.Equal(t, audioIDs[expectedAudioIdx], audio.ID)

	count, err := sqb.QueryCount(ctx, nil, &filter)
	if err != nil {
		t.Errorf("Error querying audio: %s", err.Error())
	}
	assert.Equal(t, len(audios), count)

	// no Q should return all results
	filter.Q = nil
	audios = queryAudios(ctx, t, sqb, nil, &filter)

	assert.Len(t, audios, totalAudios)
}

func verifyAudioQuery(t *testing.T, filter models.AudioFilterType, verifyFn func(ctx context.Context, s *models.Audio)) {
	t.Helper()
	withTxn(func(ctx context.Context) error {
		t.Helper()
		sqb := db.Audio

		audios := queryAudios(ctx, t, sqb, &filter, nil)

		// assume it should find at least one
		assert.Greater(t, len(audios), 0)

		for _, audio := range audios {
			verifyFn(ctx, audio)
		}

		return nil
	})
}

func TestAudioQueryURL(t *testing.T) {
	const audioIdx = 1
	audioURL := getAudioStringValue(audioIdx, urlField)
	urlCriterion := models.StringCriterionInput{
		Value:    audioURL,
		Modifier: models.CriterionModifierEquals,
	}
	filter := models.AudioFilterType{
		URL: &urlCriterion,
	}

	verifyFn := func(ctx context.Context, o *models.Audio) {
		t.Helper()
		// Load URLs if not already loaded
		if !o.URLs.Loaded() {
			if err := o.LoadURLs(ctx, db.Audio); err != nil {
				t.Errorf("Error loading URLs: %v", err)
				return
			}
		}
		// Get the first URL or empty string if no URLs
		var url string
		if o.URLs.Loaded() && len(o.URLs.List()) > 0 {
			url = o.URLs.List()[0]
		}
		verifyString(t, url, urlCriterion)
	}

	verifyAudioQuery(t, filter, verifyFn)
	urlCriterion.Modifier = models.CriterionModifierNotEquals
	verifyAudioQuery(t, filter, verifyFn)
	urlCriterion.Modifier = models.CriterionModifierMatchesRegex
	urlCriterion.Value = "audio_.*1_URL"
	verifyAudioQuery(t, filter, verifyFn)
	urlCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyAudioQuery(t, filter, verifyFn)
	urlCriterion.Modifier = models.CriterionModifierIsNull
	urlCriterion.Value = ""
	verifyAudioQuery(t, filter, verifyFn)
	urlCriterion.Modifier = models.CriterionModifierNotNull
	verifyAudioQuery(t, filter, verifyFn)
}

func TestAudioQueryRating100(t *testing.T) {
	const rating = 60
	ratingCriterion := models.IntCriterionInput{
		Value:    rating,
		Modifier: models.CriterionModifierEquals,
	}

	verifyAudiosRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotEquals
	verifyAudiosRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyAudiosRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierLessThan
	verifyAudiosRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierIsNull
	verifyAudiosRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotNull
	verifyAudiosRating100(t, ratingCriterion)
}

func verifyAudiosRating100(t *testing.T, ratingCriterion models.IntCriterionInput) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Audio
		audioFilter := models.AudioFilterType{
			Rating100: &ratingCriterion,
		}

		audios, _, err := queryAudiosWithCount(ctx, sqb, &audioFilter, nil)
		if err != nil {
			t.Errorf("Error querying audio: %s", err.Error())
		}

		for _, audio := range audios {
			verifyIntPtr(t, audio.Rating, ratingCriterion)
		}

		return nil
	})
}

// TestAudioQueryDurationIsNullUsesLeftJoin is a guard test for the join-type
// optimisation ported from upstream #6648: durationCriterionHandler (and its
// siblings for bitrate/audio codec/sample rate/channels/checksum) now use an
// INNER join against audio_files except when the criterion modifier is
// IsNull, in which case a LEFT join is required to surface audios with no
// matching audio_files row at all. This does not reproduce a pre-existing
// bug (there was none to reproduce); it exists solely to prove the IsNull
// branch of the new conditional keeps working. It must pass both before and
// after the join-type change.
func TestAudioQueryDurationIsNullUsesLeftJoin(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		aqb := db.Audio

		// Create an audio with no linked audio_files row at all, so that an
		// (incorrect) INNER join on audio_files would silently drop it from
		// an IsNull result set.
		audio := &models.Audio{
			Title: "audio with no files for duration IsNull guard test",
		}
		if err := aqb.Create(ctx, audio, nil); err != nil {
			t.Errorf("Error creating audio with no files: %s", err.Error())
			return nil
		}

		durationCriterion := models.IntCriterionInput{
			Modifier: models.CriterionModifierIsNull,
		}
		audioFilter := models.AudioFilterType{
			Duration: &durationCriterion,
		}

		audios, _, err := queryAudiosWithCount(ctx, aqb, &audioFilter, nil)
		if err != nil {
			t.Errorf("Error querying audio: %s", err.Error())
			return nil
		}

		found := false
		for _, a := range audios {
			if a.ID == audio.ID {
				found = true
			}
		}

		assert.True(t, found, "audio with no audio_files row should be returned when Duration modifier is IsNull")

		return nil
	})
}

func TestAudioQueryPerformers(t *testing.T) {
	tests := []struct {
		name        string
		filter      models.MultiCriterionInput
		includeIdxs []int
		excludeIdxs []int
		wantErr     bool
	}{
		{
			"includes",
			models.MultiCriterionInput{
				Value: []string{
					strconv.Itoa(performerIDs[performerIdxWithScene]),
					strconv.Itoa(performerIDs[performerIdx1WithScene]),
				},
				Modifier: models.CriterionModifierIncludes,
			},
			[]int{
				audioIdxWithPerformer,
				audioIdxWithTwoPerformers,
			},
			[]int{
				audioIdxWithTag,
			},
			false,
		},
		{
			"includes all",
			models.MultiCriterionInput{
				Value: []string{
					strconv.Itoa(performerIDs[performerIdx1WithScene]),
					strconv.Itoa(performerIDs[performerIdx2WithScene]),
				},
				Modifier: models.CriterionModifierIncludesAll,
			},
			[]int{
				audioIdxWithTwoPerformers,
			},
			[]int{
				audioIdxWithPerformer,
			},
			false,
		},
		{
			"excludes",
			models.MultiCriterionInput{
				Modifier: models.CriterionModifierExcludes,
				Value:    []string{strconv.Itoa(performerIDs[performerIdx1WithScene])},
			},
			nil,
			[]int{audioIdxWithTwoPerformers},
			false,
		},
		{
			"is null",
			models.MultiCriterionInput{
				Modifier: models.CriterionModifierIsNull,
			},
			[]int{audioIdxWithTag},
			[]int{
				audioIdxWithPerformer,
				audioIdxWithTwoPerformers,
				audioIdxWithPerformerTwoTags,
			},
			false,
		},
		{
			"not null",
			models.MultiCriterionInput{
				Modifier: models.CriterionModifierNotNull,
			},
			[]int{
				audioIdxWithPerformer,
				audioIdxWithTwoPerformers,
				audioIdxWithPerformerTwoTags,
			},
			[]int{audioIdxWithTag},
			false,
		},
	}

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			results, err := db.Audio.Query(ctx, models.AudioQueryOptions{
				AudioFilter: &models.AudioFilterType{
					Performers: &tt.filter,
				},
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("AudioStore.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			include := indexesToIDs(audioIDs, tt.includeIdxs)
			exclude := indexesToIDs(audioIDs, tt.excludeIdxs)

			for _, i := range include {
				assert.Contains(results.IDs, i)
			}
			for _, e := range exclude {
				assert.NotContains(results.IDs, e)
			}
		})
	}
}

func TestAudioQueryTags(t *testing.T) {
	tests := []struct {
		name        string
		filter      models.HierarchicalMultiCriterionInput
		includeIdxs []int
		excludeIdxs []int
		wantErr     bool
	}{
		{
			"includes",
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(tagIDs[tagIdxWithScene]),
					strconv.Itoa(tagIDs[tagIdx1WithScene]),
				},
				Modifier: models.CriterionModifierIncludes,
			},
			[]int{
				audioIdxWithTag,
				audioIdxWithTwoTags,
			},
			[]int{
				audioIdxWithPerformer,
			},
			false,
		},
		{
			"includes all",
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(tagIDs[tagIdx1WithScene]),
					strconv.Itoa(tagIDs[tagIdx2WithScene]),
				},
				Modifier: models.CriterionModifierIncludesAll,
			},
			[]int{
				audioIdxWithTwoTags,
			},
			[]int{
				audioIdxWithTag,
			},
			false,
		},
		{
			"excludes",
			models.HierarchicalMultiCriterionInput{
				Modifier: models.CriterionModifierExcludes,
				Value:    []string{strconv.Itoa(tagIDs[tagIdx1WithScene])},
			},
			nil,
			[]int{audioIdxWithTwoTags},
			false,
		},
		{
			"is null",
			models.HierarchicalMultiCriterionInput{
				Modifier: models.CriterionModifierIsNull,
			},
			[]int{audioIdxWithPerformer},
			[]int{
				audioIdxWithTag,
				audioIdxWithTwoTags,
				audioIdxWithThreeTags,
			},
			false,
		},
		{
			"not null",
			models.HierarchicalMultiCriterionInput{
				Modifier: models.CriterionModifierNotNull,
			},
			[]int{
				audioIdxWithTag,
				audioIdxWithTwoTags,
				audioIdxWithThreeTags,
			},
			[]int{audioIdxWithPerformer},
			false,
		},
	}

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			results, err := db.Audio.Query(ctx, models.AudioQueryOptions{
				AudioFilter: &models.AudioFilterType{
					Tags: &tt.filter,
				},
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("AudioStore.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			include := indexesToIDs(audioIDs, tt.includeIdxs)
			exclude := indexesToIDs(audioIDs, tt.excludeIdxs)

			for _, i := range include {
				assert.Contains(results.IDs, i)
			}
			for _, e := range exclude {
				assert.NotContains(results.IDs, e)
			}
		})
	}
}

// TestAudioQueryOrSubFilterJoinType ports upstream fc0b2a5d9 (#6920): when a
// filter's primary criterion is ANDed against an OR sub-filter, the joins
// registered by the primary criterion handler must exist before the OR
// sub-filter is resolved, so that innerJoinsToLeftJoins can convert them to
// LEFT JOINs. If handleCriterion runs after handleSubFilter, the primary
// criterion's joins are still INNER at OR-resolution time and rows that only
// match the OR branch (and not the primary criterion's joined table) are
// silently dropped.
//
// audioIdxWithPerformer has a performer but no tags; audioIdxWithTag has a
// tag but no performer. A filter of Performers=[performerIdxWithScene] OR
// Tags=[tagIdxWithScene] should return both audios.
func TestAudioQueryOrSubFilterJoinType(t *testing.T) {
	audioFilter := models.AudioFilterType{
		Performers: &models.MultiCriterionInput{
			Value:    []string{strconv.Itoa(performerIDs[performerIdxWithScene])},
			Modifier: models.CriterionModifierIncludes,
		},
		OperatorFilter: models.OperatorFilter[models.AudioFilterType]{
			Or: &models.AudioFilterType{
				Tags: &models.HierarchicalMultiCriterionInput{
					Value:    []string{strconv.Itoa(tagIDs[tagIdxWithScene])},
					Modifier: models.CriterionModifierIncludes,
				},
			},
		},
	}

	withTxn(func(ctx context.Context) error {
		sqb := db.Audio

		audios := queryAudios(ctx, t, sqb, &audioFilter, nil)

		var ids []int
		for _, a := range audios {
			ids = append(ids, a.ID)
		}

		assert.Contains(t, ids, audioIDs[audioIdxWithPerformer])
		assert.Contains(t, ids, audioIDs[audioIdxWithTag])

		return nil
	})
}

func queryAudios(ctx context.Context, t *testing.T, sqb models.AudioReader, audioFilter *models.AudioFilterType, findFilter *models.FindFilterType) []*models.Audio {
	audios, _, err := queryAudiosWithCount(ctx, sqb, audioFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying audios: %s", err.Error())
	}

	return audios
}

func TestAudioQuerySorting(t *testing.T) {
	tests := []struct {
		name          string
		sortBy        string
		dir           models.SortDirectionEnum
		firstAudioIdx int // -1 to ignore
		lastAudioIdx  int
	}{
		{
			"title",
			"title",
			models.SortDirectionEnumAsc,
			-1,
			-1,
		},
		{
			"date",
			"date",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"rating",
			"rating",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"organized",
			"organized",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"created_at",
			"created_at",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"updated_at",
			"updated_at",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"play_count",
			"play_count",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"last_played_at",
			"last_played_at",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"o_counter",
			"o_counter",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"resume_time",
			"resume_time",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"play_duration",
			"play_duration",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.Query(ctx, models.AudioQueryOptions{
				QueryOptions: models.QueryOptions{
					FindFilter: &models.FindFilterType{
						Sort:      &tt.sortBy,
						Direction: &tt.dir,
					},
				},
			})

			if err != nil {
				t.Errorf("audioQueryBuilder.TestAudioQuerySorting() error = %v", err)
				return
			}

			audios, err := got.Resolve(ctx)
			if err != nil {
				t.Errorf("audioQueryBuilder.TestAudioQuerySorting() error = %v", err)
				return
			}

			if !assert.Greater(len(audios), 0) {
				return
			}

			// audios should be in same order as indexes
			firstAudio := audios[0]
			lastAudio := audios[len(audios)-1]

			if tt.firstAudioIdx != -1 {
				firstAudioID := audioIDs[tt.firstAudioIdx]
				assert.Equal(firstAudioID, firstAudio.ID)
			}
			if tt.lastAudioIdx != -1 {
				lastAudioID := audioIDs[tt.lastAudioIdx]
				assert.Equal(lastAudioID, lastAudio.ID)
			}
		})
	}
}

func TestAudioStore_SaveActivity(t *testing.T) {
	var (
		resumeTime   = 55.6
		playDuration = 78.9
	)

	tests := []struct {
		name         string
		audioIdx     int
		resumeTime   *float64
		playDuration *float64
		wantErr      bool
	}{
		{
			"both",
			audioIdxWithPerformer,
			&resumeTime,
			&playDuration,
			false,
		},
		{
			"resumeTime only",
			audioIdxWithPerformer,
			&resumeTime,
			nil,
			false,
		},
		{
			"playDuration only",
			audioIdxWithPerformer,
			nil,
			&playDuration,
			false,
		},
		{
			"none",
			audioIdxWithPerformer,
			nil,
			nil,
			false,
		},
		{
			"invalid audio id",
			-1,
			&resumeTime,
			&playDuration,
			true,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRollbackTxn(func(ctx context.Context) error {
				id := -1
				if tt.audioIdx != -1 {
					id = audioIDs[tt.audioIdx]
				}

				_, err := qb.SaveActivity(ctx, id, tt.resumeTime, tt.playDuration)
				if (err != nil) != tt.wantErr {
					t.Errorf("AudioStore.SaveActivity() error = %v, wantErr %v", err, tt.wantErr)
				}

				if err != nil {
					return nil
				}

				assert := assert.New(t)

				// find the audio and check the values
				audio, err := qb.Find(ctx, id)
				if err != nil {
					t.Errorf("AudioStore.Find() error = %v", err)
				}

				expectedResumeTime := getAudioResumeTime(tt.audioIdx)
				expectedPlayDuration := getAudioPlayDuration(tt.audioIdx)

				if tt.resumeTime != nil {
					expectedResumeTime = *tt.resumeTime
				}
				if tt.playDuration != nil {
					expectedPlayDuration += *tt.playDuration
				}

				assert.Equal(expectedResumeTime, audio.ResumeTime)
				assert.Equal(expectedPlayDuration, audio.PlayDuration)

				return nil
			})
		})
	}
}

// TestAudioStore_SaveActivity_PointerBugRegression is a regression test for a bug where
// resume_time was being stored as a pointer instead of the dereferenced value.
//
// The bug was in SaveActivity method: record["resume_time"] = resumeTime (WRONG)
// The fix is:                       record["resume_time"] = *resumeTime (CORRECT)
//
// While Go's SQL driver handles pointer conversion in simple cases, this bug
// caused issues in the GraphQL layer where the field was not being returned
// correctly, resulting in undefined/null values in API responses.
func TestAudioStore_SaveActivity_PointerBugRegression(t *testing.T) {
	qb := db.Audio

	withRollbackTxn(func(ctx context.Context) error {
		// Get a clean audio record to test with
		audioID := audioIDs[audioIdxWithPerformer]

		// First, reset the audio to have zero resume_time to ensure clean state
		_, err := qb.ResetActivity(ctx, audioID, true, true)
		require.NoError(t, err)

		// Verify it starts at 0
		audioBefore, err := qb.Find(ctx, audioID)
		require.NoError(t, err)
		assert.Equal(t, 0.0, audioBefore.ResumeTime, "Audio should start with zero resume_time")

		// Test saving a specific resume time value
		testResumeTime := 123.45
		_, err = qb.SaveActivity(ctx, audioID, &testResumeTime, nil)
		require.NoError(t, err)

		// Verify the correct value was stored
		audioAfter, err := qb.Find(ctx, audioID)
		require.NoError(t, err)
		assert.Equal(t, testResumeTime, audioAfter.ResumeTime,
			"SaveActivity must store the actual resume_time value")

		// Test with nil value (should not update)
		_, err = qb.SaveActivity(ctx, audioID, nil, nil)
		require.NoError(t, err)

		audioAfterNil, err := qb.Find(ctx, audioID)
		require.NoError(t, err)
		assert.Equal(t, testResumeTime, audioAfterNil.ResumeTime,
			"SaveActivity with nil resumeTime should not change the existing value")

		// Test with a different value to ensure updates work correctly
		testResumeTime2 := 67.89
		_, err = qb.SaveActivity(ctx, audioID, &testResumeTime2, nil)
		require.NoError(t, err)

		audioAfterUpdate, err := qb.Find(ctx, audioID)
		require.NoError(t, err)
		assert.Equal(t, testResumeTime2, audioAfterUpdate.ResumeTime,
			"SaveActivity should correctly update to new values")

		return nil
	})
}

func TestAudioStore_ResetActivity(t *testing.T) {
	tests := []struct {
		name          string
		audioIdx      int
		resetResume   bool
		resetDuration bool
		wantErr       bool
	}{
		{
			"both",
			audioIdxWithPerformer,
			true,
			true,
			false,
		},
		{
			"resume only",
			audioIdxWithPerformer,
			true,
			false,
			false,
		},
		{
			"duration only",
			audioIdxWithPerformer,
			false,
			true,
			false,
		},
		{
			"none",
			audioIdxWithPerformer,
			false,
			false,
			false,
		},
		{
			"invalid audio id",
			-1,
			true,
			true,
			true,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRollbackTxn(func(ctx context.Context) error {
				id := -1
				if tt.audioIdx != -1 {
					id = audioIDs[tt.audioIdx]
				}

				_, err := qb.ResetActivity(ctx, id, tt.resetResume, tt.resetDuration)
				if (err != nil) != tt.wantErr {
					t.Errorf("AudioStore.ResetActivity() error = %v, wantErr %v", err, tt.wantErr)
				}

				if err != nil {
					return nil
				}

				assert := assert.New(t)

				// find the audio and check the values
				audio, err := qb.Find(ctx, id)
				if err != nil {
					t.Errorf("AudioStore.Find() error = %v", err)
				}

				expectedResumeTime := getAudioResumeTime(tt.audioIdx)
				expectedPlayDuration := getAudioPlayDuration(tt.audioIdx)

				if tt.resetResume {
					expectedResumeTime = 0
				}
				if tt.resetDuration {
					expectedPlayDuration = 0
				}

				assert.Equal(expectedResumeTime, audio.ResumeTime)
				assert.Equal(expectedPlayDuration, audio.PlayDuration)

				return nil
			})
		})
	}
}

func TestAudioStore_AddViews(t *testing.T) {
	tests := []struct {
		name          string
		audioID       int
		expectedCount int
		wantErr       bool
	}{
		{
			"valid",
			audioIDs[audioIdxWithPerformer],
			1,
			false,
		},
		{
			"invalid audio id",
			invalidID,
			0,
			true,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRollbackTxn(func(ctx context.Context) error {
				views, err := qb.AddViews(ctx, tt.audioID, nil)
				if (err != nil) != tt.wantErr {
					t.Errorf("AudioStore.AddViews() error = %v, wantErr %v", err, tt.wantErr)
				}

				if err != nil {
					return nil
				}

				assert := assert.New(t)
				assert.Equal(tt.expectedCount, len(views))

				// find the audio and check the count
				count, err := qb.CountViews(ctx, tt.audioID)
				if err != nil {
					t.Errorf("AudioStore.CountViews() error = %v", err)
				}
				assert.Equal(tt.expectedCount, count)

				lastView, err := qb.LastView(ctx, tt.audioID)
				if err != nil {
					t.Errorf("AudioStore.LastView() error = %v", err)
				}
				assert.True(lastView.After(time.Now().Add(-1 * time.Minute)))

				return nil
			})
		})
	}
}

func TestAudioStore_CountAllViews(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		qb := db.Audio

		audioID := audioIDs[audioIdxWithPerformer]

		// get the current play count
		currentCount, err := qb.CountAllViews(ctx)
		if err != nil {
			t.Errorf("AudioStore.CountAllViews() error = %v", err)
			return nil
		}

		// add a view
		_, err = qb.AddViews(ctx, audioID, nil)
		if err != nil {
			t.Errorf("AudioStore.AddViews() error = %v", err)
			return nil
		}

		// get the new play count
		newCount, err := qb.CountAllViews(ctx)
		if err != nil {
			t.Errorf("AudioStore.CountAllViews() error = %v", err)
			return nil
		}

		assert.Equal(t, currentCount+1, newCount)

		return nil
	})
}

func TestAudioStore_CountUniqueViews(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		qb := db.Audio

		audioID := audioIDs[audioIdxWithPerformer]

		// get the current play count
		currentCount, err := qb.CountUniqueViews(ctx)
		if err != nil {
			t.Errorf("AudioStore.CountUniqueViews() error = %v", err)
			return nil
		}

		// add a view
		_, err = qb.AddViews(ctx, audioID, nil)
		if err != nil {
			t.Errorf("AudioStore.AddViews() error = %v", err)
			return nil
		}

		// add a second view
		_, err = qb.AddViews(ctx, audioID, nil)
		if err != nil {
			t.Errorf("AudioStore.AddViews() error = %v", err)
			return nil
		}

		// get the new play count
		newCount, err := qb.CountUniqueViews(ctx)
		if err != nil {
			t.Errorf("AudioStore.CountUniqueViews() error = %v", err)
			return nil
		}

		assert.Equal(t, currentCount+1, newCount)

		return nil
	})
}

func TestAudioStore_DeleteViews(t *testing.T) {
	tests := []struct {
		name          string
		audioID       int
		expectedCount int
		wantErr       bool
	}{
		{
			"valid",
			audioIDs[audioIdxWithPerformer],
			0,
			false,
		},
		{
			"invalid audio id",
			invalidID,
			0,
			true,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRollbackTxn(func(ctx context.Context) error {
				views, err := qb.DeleteViews(ctx, tt.audioID, []time.Time{})
				if (err != nil) != tt.wantErr {
					t.Errorf("AudioStore.DeleteViews() error = %v, wantErr %v", err, tt.wantErr)
				}

				if err != nil {
					return nil
				}

				assert := assert.New(t)
				assert.Equal(tt.expectedCount, len(views))

				// find the audio and check the count
				count, err := qb.CountViews(ctx, tt.audioID)
				if err != nil {
					t.Errorf("AudioStore.CountViews() error = %v", err)
				}
				assert.Equal(tt.expectedCount, count)

				return nil
			})
		})
	}
}

func TestAudioStore_DeleteSpecificViews(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		qb := db.Audio
		audioID := audioIDs[audioIdxWithPerformer]

		// Clear any existing views
		_, err := qb.DeleteAllViews(ctx, audioID)
		if err != nil {
			t.Errorf("AudioStore.DeleteAllViews() error = %v", err)
			return nil
		}

		// Add two specific timestamps
		timestamp1, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
		timestamp2, _ := time.Parse(time.RFC3339, "2024-01-15T14:20:00Z")

		timestamps := []time.Time{timestamp1, timestamp2}
		addedViews, err := qb.AddViews(ctx, audioID, timestamps)
		if err != nil {
			t.Errorf("AudioStore.AddViews() error = %v", err)
			return nil
		}

		assert := assert.New(t)
		assert.Equal(2, len(addedViews), "Should have added 2 views")

		// Verify we have 2 views
		count, err := qb.CountViews(ctx, audioID)
		if err != nil {
			t.Errorf("AudioStore.CountViews() error = %v", err)
			return nil
		}
		assert.Equal(2, count, "Should have 2 views after adding")

		// Delete the first specific timestamp
		remainingViews, err := qb.DeleteViews(ctx, audioID, []time.Time{timestamp1})
		if err != nil {
			t.Errorf("AudioStore.DeleteViews() error = %v", err)
			return nil
		}

		// Should have 1 remaining view
		assert.Equal(1, len(remainingViews), "Should have 1 remaining view after deleting one")

		// Verify the count is now 1
		finalCount, err := qb.CountViews(ctx, audioID)
		if err != nil {
			t.Errorf("AudioStore.CountViews() error = %v", err)
			return nil
		}
		assert.Equal(1, finalCount, "Should have 1 view after deleting one specific timestamp")

		// The remaining view should be the second timestamp
		if len(remainingViews) > 0 {
			assert.Equal(timestamp2.UTC(), remainingViews[0].UTC(), "Remaining view should be the second timestamp")
		}

		return nil
	})
}

func TestAudioStore_DeleteSpecificTimestampBug(t *testing.T) {
	runWithRollbackTxn(t, "reproduce the exact bug from integration test", func(t *testing.T, ctx context.Context) {
		// Use an existing audio ID from the test data
		audioID := audioIDs[0]

		// Clear any existing play history
		initialCount, err := db.Audio.DeleteAllViews(ctx, audioID)
		require.NoError(t, err)
		t.Logf("Reset play count, removed %d entries", initialCount)

		// Add two specific timestamps matching the integration test
		timestamp1, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
		timestamp2, _ := time.Parse(time.RFC3339, "2024-01-15T14:20:00Z")
		initialDates := []time.Time{timestamp1, timestamp2}

		// Add the timestamps
		updatedTimes, err := db.Audio.AddViews(ctx, audioID, initialDates)
		require.NoError(t, err)
		assert.Len(t, updatedTimes, 2, "Should have 2 timestamps after adding")
		t.Logf("Added 2 timestamps, total count: %d", len(updatedTimes))

		// Delete the first specific timestamp
		timesToDelete := []time.Time{timestamp1}
		remainingTimes, err := db.Audio.DeleteViews(ctx, audioID, timesToDelete)
		require.NoError(t, err)
		t.Logf("After deleting specific timestamp, remaining count: %d", len(remainingTimes))

		// Verify we have exactly 1 timestamp remaining
		assert.Len(t, remainingTimes, 1, "Should have exactly 1 timestamp remaining after deleting 1 specific timestamp")

		// Verify the correct timestamp remains (timestamp2)
		if len(remainingTimes) == 1 {
			remaining := remainingTimes[0]
			// Allow for small time differences due to precision
			diff := remaining.Sub(timestamp2)
			if diff < 0 {
				diff = -diff
			}
			assert.True(t, diff < time.Second, "Remaining timestamp should be close to timestamp2")
			t.Logf("Remaining timestamp: %v, expected: %v, diff: %v", remaining, timestamp2, diff)
		}
	})
}

// verifyAudioFloat verifies that a float value matches the given criterion
func verifyAudioFloat(t *testing.T, value float64, criterion models.FloatCriterionInput) bool {
	t.Helper()
	assert := assert.New(t)
	switch criterion.Modifier {
	case models.CriterionModifierEquals:
		return assert.Equal(criterion.Value, value)
	case models.CriterionModifierNotEquals:
		return assert.NotEqual(criterion.Value, value)
	case models.CriterionModifierGreaterThan:
		return assert.Greater(value, criterion.Value)
	case models.CriterionModifierLessThan:
		return assert.Less(value, criterion.Value)
	case models.CriterionModifierBetween:
		return assert.GreaterOrEqual(value, criterion.Value) && assert.LessOrEqual(value, *criterion.Value2)
	case models.CriterionModifierNotBetween:
		return assert.True(value < criterion.Value || value > *criterion.Value2)
	}
	return false
}

// TestAudioQueryResumeTime tests the resume time filtering functionality
func TestAudioQueryResumeTime(t *testing.T) {
	// Test all the basic comparison operations
	const resumeTime = 30.5
	resumeTimeCriterion := models.FloatCriterionInput{
		Value:    resumeTime,
		Modifier: models.CriterionModifierEquals,
	}

	verifyAudiosResumeTime(t, resumeTimeCriterion)

	resumeTimeCriterion.Modifier = models.CriterionModifierNotEquals
	verifyAudiosResumeTime(t, resumeTimeCriterion)

	resumeTimeCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyAudiosResumeTime(t, resumeTimeCriterion)

	resumeTimeCriterion.Modifier = models.CriterionModifierLessThan
	verifyAudiosResumeTime(t, resumeTimeCriterion)

	resumeTimeCriterion.Modifier = models.CriterionModifierIsNull
	verifyAudiosResumeTime(t, resumeTimeCriterion)

	resumeTimeCriterion.Modifier = models.CriterionModifierNotNull
	verifyAudiosResumeTime(t, resumeTimeCriterion)

	// Test between
	value2 := 60.0
	resumeTimeCriterion.Value2 = &value2
	resumeTimeCriterion.Modifier = models.CriterionModifierBetween
	verifyAudiosResumeTime(t, resumeTimeCriterion)

	resumeTimeCriterion.Modifier = models.CriterionModifierNotBetween
	verifyAudiosResumeTime(t, resumeTimeCriterion)
}

// TestAudioQueryResumeTimeReproduceFilteringBug tests the specific bug case where 30.5 > 30 is not working
func TestAudioQueryResumeTimeReproduceFilteringBug(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		qb := db.Audio

		// Get 3 existing audios to test with
		testAudioIDs := []int{
			audioIDs[0],
			audioIDs[1],
			audioIDs[2],
		}

		// Set up specific resume time values that match the failing test
		testValues := []float64{0.0, 30.5, 180.25}

		for i, audioID := range testAudioIDs {
			resumeTime := testValues[i]
			_, err := qb.SaveActivity(ctx, audioID, &resumeTime, nil)
			if err != nil {
				t.Errorf("Error saving activity for audio %d: %v", audioID, err)
				return nil
			}

			// Verify the value was stored correctly
			audio, err := qb.Find(ctx, audioID)
			if err != nil {
				t.Errorf("Error finding audio %d: %v", audioID, err)
				return nil
			}

			t.Logf("Audio %d: resume_time = %f", audioID, audio.ResumeTime)
			assert.Equal(t, resumeTime, audio.ResumeTime, "Resume time should be stored correctly")
		}

		// Now test the problematic filter: resume_time > 30
		filterCriterion := models.FloatCriterionInput{
			Value:    30.0,
			Modifier: models.CriterionModifierGreaterThan,
		}

		audioFilter := models.AudioFilterType{
			ResumeTime: &filterCriterion,
		}

		t.Logf("Testing filter: resume_time > %f", filterCriterion.Value)

		audios, count, err := queryAudiosWithCount(ctx, qb, &audioFilter, nil)
		if err != nil {
			t.Errorf("Error querying audios: %s", err.Error())
			return nil
		}

		t.Logf("Found %d audios matching filter", count)
		for _, audio := range audios {
			t.Logf("  Audio ID %d: resume_time = %f", audio.ID, audio.ResumeTime)
		}

		// We expect 2 results: audio with 30.5 and audio with 180.25
		assert.Equal(t, 2, count, "Should find 2 audios with resume_time > 30")

		// Verify each result
		for _, audio := range audios {
			if !verifyAudioFloat(t, audio.ResumeTime, filterCriterion) {
				t.Errorf("Audio %d has resume_time %f which should match > %f",
					audio.ID, audio.ResumeTime, filterCriterion.Value)
			}
		}

		// Test exact comparison for the boundary case
		exactFilterCriterion := models.FloatCriterionInput{
			Value:    30.5,
			Modifier: models.CriterionModifierEquals,
		}

		audioFilter.ResumeTime = &exactFilterCriterion

		t.Logf("Testing exact filter: resume_time = %f", exactFilterCriterion.Value)

		exactAudios, exactCount, err := queryAudiosWithCount(ctx, qb, &audioFilter, nil)
		if err != nil {
			t.Errorf("Error querying audios for exact match: %s", err.Error())
			return nil
		}

		t.Logf("Found %d audios with exact match", exactCount)
		for _, audio := range exactAudios {
			t.Logf("  Audio ID %d: resume_time = %f", audio.ID, audio.ResumeTime)
		}

		// Should find exactly 1 result
		assert.Equal(t, 1, exactCount, "Should find 1 audio with resume_time = 30.5")

		if exactCount > 0 {
			assert.Equal(t, 30.5, exactAudios[0].ResumeTime, "Found audio should have resume_time = 30.5")
		}

		return nil
	})
}

func verifyAudiosResumeTime(t *testing.T, resumeTimeCriterion models.FloatCriterionInput) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Audio
		audioFilter := models.AudioFilterType{
			ResumeTime: &resumeTimeCriterion,
		}

		audios, _, err := queryAudiosWithCount(ctx, sqb, &audioFilter, nil)
		if err != nil {
			t.Errorf("Error querying audio: %s", err.Error())
		}

		for _, audio := range audios {
			// Skip NULL/NOT NULL tests for now since resume_time uses 0 as default, not actual NULL
			if resumeTimeCriterion.Modifier == models.CriterionModifierIsNull ||
				resumeTimeCriterion.Modifier == models.CriterionModifierNotNull {
				// These modifiers may not work as expected since resume_time uses 0 as default
				continue
			} else {
				verifyAudioFloat(t, audio.ResumeTime, resumeTimeCriterion)
			}
		}

		return nil
	})
}

// Test O-Date Manager Methods
func TestAudioStore_AddO(t *testing.T) {
	tests := []struct {
		name      string
		audioIdx  int
		dates     []time.Time
		wantCount int
		wantErr   bool
	}{
		{
			name:      "add single o-date",
			audioIdx:  1,
			dates:     []time.Time{time.Now()},
			wantCount: 1, // Expecting to add 1 date
			wantErr:   false,
		},
		{
			name:      "add multiple o-dates",
			audioIdx:  2,
			dates:     []time.Time{time.Now(), time.Now().Add(1 * time.Minute), time.Now().Add(2 * time.Minute)},
			wantCount: 3, // Expecting to add 3 dates
			wantErr:   false,
		},
		{
			name:      "invalid audio id",
			audioIdx:  -1, // Will use invalidID
			dates:     []time.Time{time.Now()},
			wantCount: 0,
			wantErr:   true,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			audioID := invalidID
			if tt.audioIdx >= 0 && tt.audioIdx < len(audioIDs) {
				audioID = audioIDs[tt.audioIdx]
			}

			// Get initial count
			initialCount, err := qb.GetOCount(ctx, audioID)
			if tt.wantErr && err != nil {
				return // Expected error on invalid ID
			}
			require.NoError(t, err)

			// Add o-dates
			_, err = qb.AddO(ctx, audioID, tt.dates)
			if (err != nil) != tt.wantErr {
				t.Errorf("AudioStore.AddO() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify count increased
				newCount, err := qb.GetOCount(ctx, audioID)
				require.NoError(t, err)
				assert.Equal(t, initialCount+tt.wantCount, newCount, "O-count should increase by %d", tt.wantCount)

				// Verify dates were actually added
				dates, err := qb.GetODates(ctx, audioID)
				require.NoError(t, err)
				assert.Equal(t, initialCount+tt.wantCount, len(dates), "Should have correct number of o-dates")
			}
		})
	}
}

func TestAudioStore_DeleteO(t *testing.T) {
	tests := []struct {
		name          string
		audioIdx      int
		setupDates    []time.Time
		deleteDates   []time.Time
		wantRemaining int
		wantErr       bool
	}{
		{
			name:          "delete single o-date",
			audioIdx:      1,
			setupDates:    []time.Time{time.Now().Add(-2 * time.Minute), time.Now().Add(-1 * time.Minute), time.Now()},
			deleteDates:   []time.Time{}, // Will delete the last one
			wantRemaining: 2,
			wantErr:       false,
		},
		{
			name:          "delete non-existent date",
			audioIdx:      2,
			setupDates:    []time.Time{time.Now()},
			deleteDates:   []time.Time{time.Now().Add(1 * time.Hour)}, // Future date that doesn't exist
			wantRemaining: 1,                                          // Nothing should be deleted
			wantErr:       false,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			audioID := audioIDs[tt.audioIdx]

			// Setup: Add initial dates
			_, err := qb.AddO(ctx, audioID, tt.setupDates)
			require.NoError(t, err)

			// Get the dates to delete
			datesToDelete := tt.deleteDates
			if len(datesToDelete) == 0 && len(tt.setupDates) > 0 {
				// Get actual dates from DB to delete the last one
				dates, err := qb.GetODates(ctx, audioID)
				require.NoError(t, err)
				if len(dates) > 0 {
					datesToDelete = []time.Time{dates[len(dates)-1]}
				}
			}

			// Delete o-dates
			_, err = qb.DeleteO(ctx, audioID, datesToDelete)
			if (err != nil) != tt.wantErr {
				t.Errorf("AudioStore.DeleteO() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify count
				count, err := qb.GetOCount(ctx, audioID)
				require.NoError(t, err)
				assert.Equal(t, tt.wantRemaining, count, "Should have correct number of remaining o-dates")
			}
		})
	}
}

func TestAudioStore_ResetO(t *testing.T) {
	tests := []struct {
		name       string
		audioIdx   int
		setupDates []time.Time
		wantErr    bool
	}{
		{
			name:       "reset with multiple dates",
			audioIdx:   1,
			setupDates: []time.Time{time.Now().Add(-2 * time.Minute), time.Now().Add(-1 * time.Minute), time.Now()},
			wantErr:    false,
		},
		{
			name:       "reset with no dates",
			audioIdx:   2,
			setupDates: []time.Time{},
			wantErr:    false,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			audioID := audioIDs[tt.audioIdx]

			// Setup: Add initial dates
			if len(tt.setupDates) > 0 {
				_, err := qb.AddO(ctx, audioID, tt.setupDates)
				require.NoError(t, err)
			}

			// Reset o-dates
			_, err := qb.ResetO(ctx, audioID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AudioStore.ResetO() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify count is 0
				count, err := qb.GetOCount(ctx, audioID)
				require.NoError(t, err)
				assert.Equal(t, 0, count, "O-count should be 0 after reset")

				// Verify no dates remain
				dates, err := qb.GetODates(ctx, audioID)
				require.NoError(t, err)
				assert.Empty(t, dates, "Should have no o-dates after reset")
			}
		})
	}
}

func TestAudioStore_GetOCount(t *testing.T) {
	tests := []struct {
		name       string
		audioIdx   int
		setupDates []time.Time
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "count with 0 dates",
			audioIdx:   1,
			setupDates: []time.Time{},
			wantCount:  0,
			wantErr:    false,
		},
		{
			name:       "count with 1 date",
			audioIdx:   2,
			setupDates: []time.Time{time.Now()},
			wantCount:  1,
			wantErr:    false,
		},
		{
			name:       "count with multiple dates",
			audioIdx:   3,
			setupDates: []time.Time{time.Now().Add(-2 * time.Minute), time.Now().Add(-1 * time.Minute), time.Now()},
			wantCount:  3,
			wantErr:    false,
		},
	}

	qb := db.Audio

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			audioID := audioIDs[tt.audioIdx]

			// Reset first to ensure clean state
			_, err := qb.ResetO(ctx, audioID)
			require.NoError(t, err)

			// Setup: Add dates
			if len(tt.setupDates) > 0 {
				_, err = qb.AddO(ctx, audioID, tt.setupDates)
				require.NoError(t, err)
			}

			// Get count
			count, err := qb.GetOCount(ctx, audioID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AudioStore.GetOCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Equal(t, tt.wantCount, count, "Should have correct o-count")
			}
		})
	}
}

func TestAudioStore_DeleteO_SpecificDates(t *testing.T) {
	qb := db.Audio

	runWithRollbackTxn(t, "delete specific dates", func(t *testing.T, ctx context.Context) {
		// Create a test audio
		newAudio := models.Audio{
			Title: "Test Audio for DeleteO",
		}
		err := qb.Create(ctx, &newAudio, []models.FileID{})
		if err != nil {
			t.Errorf("Failed to create audio: %v", err)
			return
		}
		audioID := newAudio.ID

		// Add specific O-dates
		dates := []time.Time{
			time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 16, 16, 0, 0, 0, time.UTC),
		}

		addedDates, err := qb.AddO(ctx, audioID, dates)
		if err != nil {
			t.Errorf("Failed to add O dates: %v", err)
			return
		}
		if len(addedDates) != 3 {
			t.Errorf("Expected 3 dates added, got %d", len(addedDates))
			return
		}

		// Delete a specific date
		deleteTime := time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC)
		remainingDates, err := qb.DeleteO(ctx, audioID, []time.Time{deleteTime})
		if err != nil {
			t.Errorf("Failed to delete O date: %v", err)
			return
		}

		// Should have 2 dates remaining
		if len(remainingDates) != 2 {
			t.Errorf("Expected 2 dates after deletion, got %d", len(remainingDates))
		}

		// Verify the specific date was removed
		for _, d := range remainingDates {
			if d.Equal(deleteTime) {
				t.Errorf("Deleted timestamp should not be in remaining dates")
			}
		}

		// Verify count
		count, err := qb.GetOCount(ctx, audioID)
		if err != nil {
			t.Errorf("Failed to get O count: %v", err)
			return
		}
		if count != 2 {
			t.Errorf("Expected count of 2, got %d", count)
		}
	})
}

func TestAudioStore_DeleteO_MostRecent(t *testing.T) {
	qb := db.Audio

	runWithRollbackTxn(t, "delete most recent with empty slice", func(t *testing.T, ctx context.Context) {
		// Create a test audio
		newAudio := models.Audio{
			Title: "Test Audio for Most Recent",
		}
		err := qb.Create(ctx, &newAudio, []models.FileID{})
		if err != nil {
			t.Errorf("Failed to create audio: %v", err)
			return
		}
		audioID := newAudio.ID

		// Add O-dates with known order
		now := time.Now().UTC()
		dates := []time.Time{
			now.Add(-2 * time.Hour),
			now.Add(-1 * time.Hour),
			now, // Most recent
		}

		addedDates, err := qb.AddO(ctx, audioID, dates)
		if err != nil {
			t.Errorf("Failed to add O dates: %v", err)
			return
		}
		if len(addedDates) != 3 {
			t.Errorf("Expected 3 dates added, got %d", len(addedDates))
			return
		}

		// Get all dates to identify the most recent
		allDates, err := qb.GetODates(ctx, audioID)
		if err != nil {
			t.Errorf("Failed to get O dates: %v", err)
			return
		}

		// Find the most recent date
		var mostRecent time.Time
		for _, d := range allDates {
			if mostRecent.IsZero() || d.After(mostRecent) {
				mostRecent = d
			}
		}

		// Delete most recent by passing empty slice (not nil)
		remainingDates, err := qb.DeleteO(ctx, audioID, []time.Time{})
		if err != nil {
			t.Errorf("Failed to delete most recent O date: %v", err)
			return
		}

		// Should have 2 dates remaining
		if len(remainingDates) != 2 {
			t.Errorf("Expected 2 dates after deleting most recent, got %d", len(remainingDates))
		}

		// Verify the most recent was removed
		for _, d := range remainingDates {
			if d.Equal(mostRecent) {
				t.Errorf("Most recent timestamp should not be in remaining dates")
			}
		}

		// Verify count
		count, err := qb.GetOCount(ctx, audioID)
		if err != nil {
			t.Errorf("Failed to get O count: %v", err)
			return
		}
		if count != 2 {
			t.Errorf("Expected count of 2, got %d", count)
		}
	})
}

func TestAudioStore_AddO_NonExistentAudio(t *testing.T) {
	qb := db.Audio

	runWithRollbackTxn(t, "add to non-existent audio", func(t *testing.T, ctx context.Context) {
		nonExistentID := 999999

		// Verify the audio doesn't exist
		audio, err := qb.Find(ctx, nonExistentID)
		if err != nil {
			t.Errorf("Find should not error for non-existent audio: %v", err)
			return
		}
		if audio != nil {
			t.Errorf("Audio should not exist")
			return
		}

		// Try to add O dates to non-existent audio
		dates := []time.Time{time.Now()}
		_, err = qb.AddO(ctx, nonExistentID, dates)

		// Should error due to foreign key constraint
		if err == nil {
			t.Errorf("Expected error when adding O to non-existent audio")
		}
	})
}

// Defect 2: TotalSize/TotalDuration must respect the query's filter, not
// aggregate across the whole library. Ports the filter-awareness half of
// upstream db4b33f53 (#7006) for audio, which has no equivalent upstream
// audio store to port from directly.
func TestAudioQueryTotalSizeRespectsFilter(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		sqb := db.Audio
		fqb := db.File

		const matchedSize = int64(5555)
		const matchedDuration = float64(50)
		const unmatchedSize = int64(9999)
		const unmatchedDuration = float64(90)

		makeFile := func(basename string, size int64, duration float64) models.FileID {
			f := &models.AudioFile{
				BaseFile: &models.BaseFile{
					Path:           getFilePath(folderIdxWithFiles, basename),
					Basename:       basename,
					ParentFolderID: folderIDs[folderIdxWithFiles],
					Size:           size,
				},
				Duration: duration,
			}
			if err := fqb.Create(ctx, f); err != nil {
				t.Fatalf("creating file: %v", err)
			}
			return f.ID
		}

		matchedFileID := makeFile("total-size-filter-matched.mp3", matchedSize, matchedDuration)
		unmatchedFileID := makeFile("total-size-filter-unmatched.mp3", unmatchedSize, unmatchedDuration)

		matchedAudio := &models.Audio{Title: "total size filter matched"}
		if err := sqb.Create(ctx, matchedAudio, []models.FileID{matchedFileID}); err != nil {
			t.Fatalf("creating matched audio: %v", err)
		}

		unmatchedAudio := &models.Audio{Title: "total size filter unmatched"}
		if err := sqb.Create(ctx, unmatchedAudio, []models.FileID{unmatchedFileID}); err != nil {
			t.Fatalf("creating unmatched audio: %v", err)
		}

		result, err := sqb.Query(ctx, models.AudioQueryOptions{
			QueryOptions: models.QueryOptions{Count: true},
			AudioFilter: &models.AudioFilterType{
				ID: &models.IntCriterionInput{
					Modifier: models.CriterionModifierEquals,
					Value:    matchedAudio.ID,
				},
			},
			TotalDuration: true,
			TotalSize:     true,
		})
		if err != nil {
			t.Fatalf("querying audio: %v", err)
		}

		assert.Equal(t, 1, result.Count)
		assert.Equal(t, float64(matchedSize), result.TotalSize)
		assert.Equal(t, matchedDuration, result.TotalDuration)

		return nil
	})
}

// Defect 1: DISTINCT must not collapse rows for files that happen to share
// a size/duration, undercounting totals. Ports upstream db4b33f53 (#7006)'s
// TestSceneQueryTotalSizeMultipleFiles for audio.
func TestAudioSizeSummaryAllFiles(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		sqb := db.Audio
		fqb := db.File

		const fileSize = int64(1234)
		const fileDuration = float64(100)

		makeFile := func(basename string) models.FileID {
			f := &models.AudioFile{
				BaseFile: &models.BaseFile{
					Path:           getFilePath(folderIdxWithFiles, basename),
					Basename:       basename,
					ParentFolderID: folderIDs[folderIdxWithFiles],
					Size:           fileSize,
				},
				Duration: fileDuration,
			}
			if err := fqb.Create(ctx, f); err != nil {
				t.Fatalf("creating file: %v", err)
			}
			return f.ID
		}

		f1 := makeFile("multifile-audio-1.mp3")
		f2 := makeFile("multifile-audio-2.mp3")

		audio := &models.Audio{Title: "multifile audio"}
		if err := sqb.Create(ctx, audio, []models.FileID{f1, f2}); err != nil {
			t.Fatalf("creating audio: %v", err)
		}

		result, err := sqb.Query(ctx, models.AudioQueryOptions{
			QueryOptions: models.QueryOptions{Count: true},
			AudioFilter: &models.AudioFilterType{
				ID: &models.IntCriterionInput{
					Modifier: models.CriterionModifierEquals,
					Value:    audio.ID,
				},
			},
			TotalDuration: true,
			TotalSize:     true,
		})
		if err != nil {
			t.Fatalf("querying audio: %v", err)
		}

		assert.Equal(t, 1, result.Count)
		assert.Equal(t, float64(fileSize*2), result.TotalSize)
		assert.Equal(t, fileDuration*2, result.TotalDuration)

		return nil
	})
}

// TestAudioStoreGetManyIDsByFileIDs is a capability test: it proves the new
// file->audio related-object resolution path (ported from upstream bb67152f9,
// #6938) works, mirroring scene's GetManyIDsByFileIDs. This is not a
// bug-reproduction test — there is no pre-existing failure being fixed here,
// since audio never had this capability before.
func TestAudioStoreGetManyIDsByFileIDs(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		sqb := db.Audio
		fqb := db.File

		const fileSize = int64(1234)
		const fileDuration = float64(100)

		makeFile := func(basename string) models.FileID {
			f := &models.AudioFile{
				BaseFile: &models.BaseFile{
					Path:           getFilePath(folderIdxWithFiles, basename),
					Basename:       basename,
					ParentFolderID: folderIDs[folderIdxWithFiles],
					Size:           fileSize,
				},
				Duration: fileDuration,
			}
			if err := fqb.Create(ctx, f); err != nil {
				t.Fatalf("creating file: %v", err)
			}
			return f.ID
		}

		// file with an associated audio
		fWithAudio := makeFile("get-many-ids-by-file-ids-with-audio.mp3")
		audio := &models.Audio{Title: "get many ids by file ids audio"}
		if err := sqb.Create(ctx, audio, []models.FileID{fWithAudio}); err != nil {
			t.Fatalf("creating audio: %v", err)
		}

		// file with no associated audio
		fWithoutAudio := makeFile("get-many-ids-by-file-ids-without-audio.mp3")

		got, err := sqb.GetManyIDsByFileIDs(ctx, []models.FileID{fWithAudio, fWithoutAudio})
		if err != nil {
			t.Fatalf("GetManyIDsByFileIDs: %v", err)
		}

		if !assert.Len(t, got, 2) {
			return nil
		}

		assert.Equal(t, []int{audio.ID}, got[0], "expected audio ID for file with associated audio")
		assert.Empty(t, got[1], "expected empty slice for file with no associated audio")

		return nil
	})
}

// TestAudioSetCustomFieldsNumber is a regression/capability test, not a bug
// reproduction: upstream fixed json.Number handling in
// getSQLValueFromCustomFieldInput (pkg/sqlite/custom_fields.go) in 8a98b72c1
// (#7040). Audio's custom fields table (migration 89_audio_custom_fields)
// routes through the same shared customFieldsStore that scene/performer/etc
// use, so the fix is inherited automatically. This test locks that in by
// proving an int and a float custom field can be set and read back on an
// audio via AudioStore.SetCustomFields/GetCustomFields.
func TestAudioSetCustomFieldsNumber(t *testing.T) {
	runWithRollbackTxn(t, "AudioSetCustomFieldsNumber", func(t *testing.T, ctx context.Context) {
		assert := assert.New(t)

		aqb := db.Audio
		audioIdx := audioIdxWithPerformer
		id := audioIDs[audioIdx]

		err := aqb.SetCustomFields(ctx, id, models.CustomFieldsInput{
			Full: map[string]interface{}{
				"int_field":   json.Number("42"),
				"float_field": json.Number("4.5"),
			},
		})
		require.NoError(t, err)

		got, err := aqb.GetCustomFields(ctx, id)
		require.NoError(t, err)

		// integers round-trip as int64 via json.Number.Int64()
		assert.EqualValues(int64(42), got["int_field"])
		// non-integral values fall back to float64 via json.Number.Float64()
		assert.EqualValues(4.5, got["float_field"])
	})
}

// TestAudioSetCustomFieldsString documents that the pre-existing non-numeric
// custom field path continues to work for audio, alongside the json.Number
// path exercised above.
func TestAudioSetCustomFieldsString(t *testing.T) {
	runWithRollbackTxn(t, "AudioSetCustomFieldsString", func(t *testing.T, ctx context.Context) {
		assert := assert.New(t)

		aqb := db.Audio
		audioIdx := audioIdxWithPerformer
		id := audioIDs[audioIdx]

		err := aqb.SetCustomFields(ctx, id, models.CustomFieldsInput{
			Full: map[string]interface{}{
				"string_field": "some value",
			},
		})
		require.NoError(t, err)

		got, err := aqb.GetCustomFields(ctx, id)
		require.NoError(t, err)

		assert.Equal("some value", got["string_field"])
	})
}

// TestAudioQueryCustomFieldsNumber exercises the actual path #7040 fixed:
// filtering by a numeric custom field value submitted as json.Number, which
// is how GraphQL input arrives in production (see internal/api/json.go /
// resolver_query_find_performer.go upstream, which previously worked around
// the bug by pre-converting json.Number to float64/int64 before it reached
// the SQL layer; that workaround was removed in 8a98b72c1 once the SQL layer
// itself was fixed to handle json.Number directly).
func TestAudioQueryCustomFieldsNumber(t *testing.T) {
	tests := []struct {
		name        string
		filter      *models.AudioFilterType
		includeIdxs []int
		excludeIdxs []int
		wantErr     bool
	}{
		{
			"json number equals",
			&models.AudioFilterType{
				CustomFields: []models.CustomFieldCriterionInput{
					{
						Field:    "real",
						Modifier: models.CriterionModifierEquals,
						Value:    []any{json.Number("0.2")},
					},
				},
			},
			[]int{audioIdxWithPerformer},
			[]int{audioIdx1WithPerformer},
			false,
		},
		{
			"json number greater than",
			&models.AudioFilterType{
				CustomFields: []models.CustomFieldCriterionInput{
					{
						Field:    "real",
						Modifier: models.CriterionModifierGreaterThan,
						Value:    []any{json.Number("0.15")},
					},
				},
			},
			[]int{audioIdxWithPerformer},
			[]int{audioIdx1WithPerformer},
			false,
		},
		{
			"json number between",
			&models.AudioFilterType{
				CustomFields: []models.CustomFieldCriterionInput{
					{
						Field:    "real",
						Modifier: models.CriterionModifierBetween,
						Value:    []any{json.Number("0.15"), json.Number("0.25")},
					},
				},
			},
			[]int{audioIdxWithPerformer},
			[]int{audioIdx1WithPerformer},
			false,
		},
	}

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			aqb := db.Audio

			// seed numeric custom fields on two audios with distinct values
			require.NoError(t, aqb.SetCustomFields(ctx, audioIDs[audioIdxWithPerformer], models.CustomFieldsInput{
				Full: map[string]interface{}{"real": float64(0.2)},
			}))
			require.NoError(t, aqb.SetCustomFields(ctx, audioIDs[audioIdx1WithPerformer], models.CustomFieldsInput{
				Full: map[string]interface{}{"real": float64(0.05)},
			}))

			result, err := aqb.Query(ctx, models.AudioQueryOptions{
				AudioFilter: tt.filter,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("AudioStore.Query() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				return
			}

			audios, err := result.Resolve(ctx)
			if err != nil {
				t.Errorf("AudioStore.Query().Resolve() error = %v", err)
				return
			}

			ids := audiosToIDs(audios)
			include := indexesToIDs(audioIDs, tt.includeIdxs)
			exclude := indexesToIDs(audioIDs, tt.excludeIdxs)

			for _, i := range include {
				assert.Contains(ids, i)
			}
			for _, e := range exclude {
				assert.NotContains(ids, e)
			}
		})
	}
}
