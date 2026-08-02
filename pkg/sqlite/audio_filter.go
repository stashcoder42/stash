package sqlite

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

type audioFilterHandler struct {
	audioFilter *models.AudioFilterType
}

func (qb *audioFilterHandler) validate() error {
	audioFilter := qb.audioFilter
	if audioFilter == nil {
		return nil
	}

	if err := validateFilterCombination(audioFilter.OperatorFilter); err != nil {
		return err
	}

	if subFilter := audioFilter.SubFilter(); subFilter != nil {
		sqb := &audioFilterHandler{audioFilter: subFilter}
		if err := sqb.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (qb *audioFilterHandler) handle(ctx context.Context, f *filterBuilder) {
	audioFilter := qb.audioFilter
	if audioFilter == nil {
		return
	}

	if err := qb.validate(); err != nil {
		f.setError(err)
		return
	}

	f.handleCriterion(ctx, qb.criterionHandler())

	sf := audioFilter.SubFilter()
	if sf != nil {
		sub := &audioFilterHandler{sf}
		handleSubFilter(ctx, sub, f, audioFilter.OperatorFilter)
	}
}

func (qb *audioFilterHandler) criterionHandler() criterionHandler {
	audioFilter := qb.audioFilter
	return compoundHandler{
		intCriterionHandler(audioFilter.ID, "audios.id", nil),
		criterionHandlerFunc(func(ctx context.Context, f *filterBuilder) {
			if audioFilter.Checksum != nil {
				qb.addAudioFilesTable(f, joinTypeLeft)
				f.addLeftJoin(fingerprintTable, "fingerprints_md5", "audios_files.file_id = fingerprints_md5.file_id AND fingerprints_md5.type = 'md5'")
			}

			stringCriterionHandler(audioFilter.Checksum, "fingerprints_md5.fingerprint")(ctx, f)
		}),
		stringCriterionHandler(audioFilter.Title, "audios.title"),
		stringCriterionHandler(audioFilter.Details, "audios.details"),
		qb.urlCriterionHandler(audioFilter.URL),

		pathCriterionHandler(audioFilter.Path, "folders.path", "files.basename", qb.addFoldersTable),
		qb.fileCountCriterionHandler(audioFilter.FileCount),
		intCriterionHandler(audioFilter.Rating100, "audios.rating", nil),
		boolCriterionHandler(audioFilter.Organized, "audios.organized", nil),
		qb.oCountCriterionHandler(audioFilter.OCounter),
		&dateCriterionHandler{audioFilter.Date, "audios.date", nil},

		floatCriterionHandler(audioFilter.ResumeTime, "audios.resume_time", nil),
		floatCriterionHandler(audioFilter.PlayDuration, "audios.play_duration", nil),
		qb.playCountCriterionHandler(audioFilter.PlayCount),
		qb.lastPlayedAtCriterionHandler(audioFilter.LastPlayedAt),
		qb.durationCriterionHandler(audioFilter.Duration),
		qb.bitrateCriterionHandler(audioFilter.Bitrate),
		qb.audioCodecCriterionHandler(audioFilter.AudioCodec),
		qb.sampleRateCriterionHandler(audioFilter.SampleRate),
		qb.channelsCriterionHandler(audioFilter.Channels),

		qb.missingCriterionHandler(audioFilter.IsMissing),

		qb.tagsCriterionHandler(audioFilter.Tags),
		qb.tagCountCriterionHandler(audioFilter.TagCount),
		qb.performersCriterionHandler(audioFilter.Performers),
		qb.performerCountCriterionHandler(audioFilter.PerformerCount),
		qb.performerAgeCriterionHandler(audioFilter.PerformerAge),
		qb.performerTagsCriterionHandler(audioFilter.PerformerTags),
		qb.performerFavoriteCriterionHandler(audioFilter.PerformerFavorite),
		&timestampCriterionHandler{audioFilter.CreatedAt, "audios.created_at", nil},
		&timestampCriterionHandler{audioFilter.UpdatedAt, "audios.updated_at", nil},

		&customFieldsFilterHandler{
			table: audiosCustomFieldsTable.GetTable(),
			fkCol: audioIDColumn,
			c:     audioFilter.CustomFields,
			idCol: "audios.id",
		},

		&relatedFilterHandler{
			relatedIDCol:   "performers_join.performer_id",
			relatedRepo:    performerRepository.repository,
			relatedHandler: &performerFilterHandler{audioFilter.PerformersFilter},
			joinFn: func(f *filterBuilder) {
				audioRepository.performers.innerJoin(f, "performers_join", "audios.id")
			},
		},

		&relatedFilterHandler{
			relatedIDCol:   "audio_tag.tag_id",
			relatedRepo:    tagRepository.repository,
			relatedHandler: &tagFilterHandler{audioFilter.TagsFilter},
			joinFn: func(f *filterBuilder) {
				audioRepository.tags.innerJoin(f, "audio_tag", "audios.id")
			},
		},
	}
}

func (qb *audioFilterHandler) addAudioFilesTable(f *filterBuilder, joinType joinType) {
	f.addJoin(joinType, audioFilesTable, "", "audios_files.audio_id = audios.id")
}

func (qb *audioFilterHandler) addFilesTable(f *filterBuilder, joinType joinType) {
	qb.addAudioFilesTable(f, joinType)
	f.addJoin(joinType, fileTable, "", "audios_files.file_id = files.id")
}

func (qb *audioFilterHandler) addFoldersTable(f *filterBuilder, joinType joinType) {
	qb.addFilesTable(f, joinType)
	f.addJoin(joinType, folderTable, "", "files.parent_folder_id = folders.id")
}

func (qb *audioFilterHandler) fileCountCriterionHandler(fileCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: audioTable,
		joinTable:    audioFilesTable,
		primaryFK:    audioIDColumn,
	}

	return h.handler(fileCount)
}

func (qb *audioFilterHandler) playCountCriterionHandler(count *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: audioTable,
		joinTable:    audiosViewDatesTable,
		primaryFK:    audioIDColumn,
	}

	return h.handler(count)
}

func (qb *audioFilterHandler) durationCriterionHandler(duration *models.IntCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if duration != nil {
			qb.addAudioFilesTable(f, joinTypeLeft)
			f.addLeftJoin("audio_files", "af_duration", "audios_files.file_id = af_duration.file_id")
			intCriterionHandler(duration, "af_duration.duration", nil)(ctx, f)
		}
	}
}

func (qb *audioFilterHandler) bitrateCriterionHandler(bitrate *models.IntCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if bitrate != nil {
			qb.addAudioFilesTable(f, joinTypeLeft)
			f.addLeftJoin("audio_files", "af_bitrate", "audios_files.file_id = af_bitrate.file_id")
			intCriterionHandler(bitrate, "af_bitrate.bitrate", nil)(ctx, f)
		}
	}
}

func (qb *audioFilterHandler) audioCodecCriterionHandler(audioCodec *models.StringCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if audioCodec != nil {
			qb.addAudioFilesTable(f, joinTypeLeft)
			f.addLeftJoin("audio_files", "af_codec", "audios_files.file_id = af_codec.file_id")
			stringCriterionHandler(audioCodec, "af_codec.audio_codec")(ctx, f)
		}
	}
}

func (qb *audioFilterHandler) sampleRateCriterionHandler(sampleRate *models.IntCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if sampleRate != nil {
			qb.addAudioFilesTable(f, joinTypeLeft)
			f.addLeftJoin("audio_files", "af_sample_rate", "audios_files.file_id = af_sample_rate.file_id")
			intCriterionHandler(sampleRate, "af_sample_rate.sample_rate", nil)(ctx, f)
		}
	}
}

func (qb *audioFilterHandler) channelsCriterionHandler(channels *models.IntCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if channels != nil {
			qb.addAudioFilesTable(f, joinTypeLeft)
			f.addLeftJoin("audio_files", "af_channels", "audios_files.file_id = af_channels.file_id")
			intCriterionHandler(channels, "af_channels.channels", nil)(ctx, f)
		}
	}
}

func (qb *audioFilterHandler) missingCriterionHandler(isMissing *string) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if isMissing != nil && *isMissing != "" {
			switch *isMissing {
			case "performers":
				audioRepository.performers.join(f, joinTypeLeft, "performers_join", "audios.id")
				f.addWhere("performers_join.audio_id IS NULL")
			case "tags":
				audioRepository.tags.join(f, joinTypeLeft, "tags_join", "audios.id")
				f.addWhere("tags_join.audio_id IS NULL")
			case "cover":
				f.addWhere("audios.cover_blob IS NULL")
			case "url":
				audiosURLsTableMgr.join(f, joinTypeLeft, "", "audios.id")
				f.addWhere("audio_urls.url IS NULL")
			default:
				if err := validateIsMissing(*isMissing, []string{
					"date", "details", "title", "rating",
				}); err != nil {
					f.setError(err)
					return
				}
				f.addWhere("(audios." + *isMissing + " IS NULL OR TRIM(audios." + *isMissing + ") = '')")
			}
		}
	}
}

func (qb *audioFilterHandler) getMultiCriterionHandlerBuilder(foreignTable, joinTable, foreignFK string, addJoinsFunc func(f *filterBuilder, joinType joinType)) multiCriterionHandlerBuilder {
	return multiCriterionHandlerBuilder{
		primaryTable: audioTable,
		foreignTable: foreignTable,
		joinTable:    joinTable,
		primaryFK:    audioIDColumn,
		foreignFK:    foreignFK,
		addJoinsFunc: addJoinsFunc,
	}
}

func (qb *audioFilterHandler) tagsCriterionHandler(tags *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	h := joinedHierarchicalMultiCriterionHandlerBuilder{
		primaryTable: audioTable,
		foreignTable: tagTable,
		foreignFK:    "tag_id",

		relationsTable: "tags_relations",
		joinAs:         "audio_tag",
		joinTable:      audioTagsTable,
		primaryFK:      audioIDColumn,
	}

	return h.handler(tags)
}

func (qb *audioFilterHandler) tagCountCriterionHandler(tagCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: audioTable,
		joinTable:    audioTagsTable,
		primaryFK:    audioIDColumn,
	}

	return h.handler(tagCount)
}

func (qb *audioFilterHandler) performersCriterionHandler(performers *models.MultiCriterionInput) criterionHandlerFunc {
	addJoinsFunc := func(f *filterBuilder, joinType joinType) {
		audioRepository.performers.join(f, joinType, "", "audios.id")
		f.addJoin(joinType, performerTable, "", "audio_performers.performer_id = performers.id")
	}
	h := qb.getMultiCriterionHandlerBuilder(performerTable, audioPerformersTable, performerIDColumn, addJoinsFunc)
	return h.handler(performers)
}

func (qb *audioFilterHandler) performerCountCriterionHandler(performerCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: audioTable,
		joinTable:    audioPerformersTable,
		primaryFK:    audioIDColumn,
	}

	return h.handler(performerCount)
}

func (qb *audioFilterHandler) performerFavoriteCriterionHandler(performerfavorite *bool) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if performerfavorite != nil {
			f.addInnerJoin(audioPerformersTable, "", "audios.id = audio_performers.audio_id")

			if *performerfavorite {
				// contains at least one favorite
				f.addInnerJoin(performerTable, "", "audio_performers.performer_id = performers.id")
				f.addWhere("performers.favorite = 1")
			} else {
				// contains zero favorites
				f.addLeftJoin(performerTable, "performers_favorite", "audio_performers.performer_id = performers_favorite.id AND performers_favorite.favorite = 1")
				f.addWhere("performers_favorite.id IS NULL")
			}
		}
	}
}

func (qb *audioFilterHandler) performerTagsCriterionHandler(tags *models.HierarchicalMultiCriterionInput) criterionHandler {
	return &joinedPerformerTagsHandler{
		criterion:      tags,
		primaryTable:   audioTable,
		joinTable:      audioPerformersTable,
		joinPrimaryKey: audioIDColumn,
	}
}

func (qb *audioFilterHandler) oCountCriterionHandler(count *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: audioTable,
		joinTable:    audiosODatesTable,
		primaryFK:    audioIDColumn,
	}

	return h.handler(count)
}

func (qb *audioFilterHandler) lastPlayedAtCriterionHandler(lastPlayedAt *models.TimestampCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if lastPlayedAt != nil {
			f.addLeftJoin(
				fmt.Sprintf("(SELECT %s, MAX(%s) as last_played_at FROM %s GROUP BY %s)", audioIDColumn, audioViewDateColumn, audiosViewDatesTable, audioIDColumn),
				"audio_last_view",
				fmt.Sprintf("audio_last_view.%s = audios.id", audioIDColumn),
			)
			h := timestampCriterionHandler{lastPlayedAt, "IFNULL(last_played_at, datetime(0))", nil}
			h.handle(ctx, f)
		}
	}
}

func (qb *audioFilterHandler) urlCriterionHandler(url *models.StringCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if url != nil {
			f.addLeftJoin(audiosURLsTable, "", "audio_urls.audio_id = audios.id")
			stringCriterionHandler(url, "audio_urls.url")(ctx, f)
		}
	}
}

func (qb *audioFilterHandler) performerAgeCriterionHandler(performerAge *models.IntCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if performerAge != nil {
			f.addInnerJoin("audio_performers", "", "audios.id = audio_performers.audio_id")
			f.addInnerJoin("performers", "", "audio_performers.performer_id = performers.id")

			f.addWhere("audios.date != '' AND performers.birthdate != ''")
			f.addWhere("audios.date IS NOT NULL AND performers.birthdate IS NOT NULL")

			ageCalc := "cast(strftime('%Y.%m%d', audios.date) - strftime('%Y.%m%d', performers.birthdate) as int)"
			whereClause, args := getIntWhereClause(ageCalc, performerAge.Modifier, performerAge.Value, performerAge.Value2)
			f.addWhere(whereClause, args...)
		}
	}
}
