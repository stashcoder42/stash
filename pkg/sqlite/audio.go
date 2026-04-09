package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"
	"gopkg.in/guregu/null.v4/zero"

	"github.com/stashapp/stash/pkg/models"
)

const (
	audioTable           = "audios"
	audioFilesTable      = "audios_files" // Join table between audios and files
	audiosURLsTable      = "audio_urls"
	audioIDColumn        = "audio_id"
	audioURLColumn       = "url"
	audioPerformersTable = "audio_performers"
	audioTagsTable       = "audio_tags"
	audiosViewDatesTable = "audios_view_dates"
	audioViewDateColumn  = "view_date"
	audiosODatesTable    = "audios_o_dates"
	audioODateColumn     = "o_date"
	audioCoverBlobColumn = "cover_blob"
)

type audioRow struct {
	ID            int         `db:"id" goqu:"skipinsert"`
	Title         zero.String `db:"title"`
	Date          NullDate    `db:"date"`
	DatePrecision null.Int    `db:"date_precision"`
	Details       zero.String `db:"details"`
	Rating        null.Int    `db:"rating"`
	Organized     bool        `db:"organized"`
	ResumeTime    float64     `db:"resume_time"`
	PlayDuration  float64     `db:"play_duration"`
	CreatedAt     Timestamp   `db:"created_at"`
	UpdatedAt     Timestamp   `db:"updated_at"`

	// not used in resolutions or updates
	CoverBlob zero.String `db:"cover_blob"`
}

func (r *audioRow) fromAudio(o models.Audio) {
	r.ID = o.ID
	r.Title = zero.StringFrom(o.Title)
	r.Date = NullDateFromDatePtr(o.Date)
	r.DatePrecision = datePrecisionFromDatePtr(o.Date)
	r.Details = zero.StringFrom(o.Details)
	r.Rating = intFromPtr(o.Rating)
	r.Organized = o.Organized
	r.ResumeTime = o.ResumeTime
	r.PlayDuration = o.PlayDuration
	r.CreatedAt = Timestamp{Timestamp: o.CreatedAt}
	r.UpdatedAt = Timestamp{Timestamp: o.UpdatedAt}
}

type audioQueryRow struct {
	audioRow
	PrimaryFileID         null.Int    `db:"primary_file_id"`
	PrimaryFileFolderPath zero.String `db:"primary_file_folder_path"`
	PrimaryFileBasename   zero.String `db:"primary_file_basename"`
	PrimaryFileChecksum   zero.String `db:"primary_file_checksum"`
}

func (r *audioQueryRow) resolve() *models.Audio {
	ret := &models.Audio{
		ID:           r.ID,
		Title:        r.Title.String,
		Date:         r.Date.DatePtr(r.DatePrecision),
		Details:      r.Details.String,
		Rating:       nullIntPtr(r.Rating),
		Organized:    r.Organized,
		ResumeTime:   r.ResumeTime,
		PlayDuration: r.PlayDuration,

		PrimaryFileID: nullIntFileIDPtr(r.PrimaryFileID),
		Checksum:      r.PrimaryFileChecksum.String,

		CreatedAt: r.CreatedAt.Timestamp,
		UpdatedAt: r.UpdatedAt.Timestamp,
	}

	if r.PrimaryFileFolderPath.Valid && r.PrimaryFileBasename.Valid {
		ret.Path = filepath.Join(r.PrimaryFileFolderPath.String, r.PrimaryFileBasename.String)
	}

	return ret
}

type audioRowRecord struct {
	updateRecord
}

func (r *audioRowRecord) fromPartial(o models.AudioPartial) {
	r.setNullString("title", o.Title)
	r.setNullDate("date", "date_precision", o.Date)
	r.setNullString("details", o.Details)
	r.setNullInt("rating", o.Rating)
	r.setBool("organized", o.Organized)
	r.setNullFloat64("resume_time", o.ResumeTime)
	r.setNullFloat64("play_duration", o.PlayDuration)
	r.setTimestamp("created_at", o.CreatedAt)
	r.setTimestamp("updated_at", o.UpdatedAt)
}

type audioRepositoryType struct {
	repository
	performers joinRepository
	tags       joinRepository
	files      filesRepository
}

func (r *audioRepositoryType) addAudioFilesTable(f *filterBuilder) {
	f.addLeftJoin(audioFilesTable, "", "audios_files.audio_id = audios.id")
}

func (r *audioRepositoryType) addFilesTable(f *filterBuilder) {
	r.addAudioFilesTable(f)
	f.addLeftJoin(fileTable, "", "audios_files.file_id = files.id")
}

func (r *audioRepositoryType) addFoldersTable(f *filterBuilder) {
	r.addFilesTable(f)
	f.addLeftJoin(folderTable, "", "files.parent_folder_id = folders.id")
}

var (
	audioRepository = audioRepositoryType{
		repository: repository{
			tableName: audioTable,
			idColumn:  idColumn,
		},

		performers: joinRepository{
			repository: repository{
				tableName: audioPerformersTable,
				idColumn:  audioIDColumn,
			},
			fkColumn: performerIDColumn,
		},

		files: filesRepository{
			repository: repository{
				tableName: audioFilesTable,
				idColumn:  audioIDColumn,
			},
		},

		tags: joinRepository{
			repository: repository{
				tableName: audioTagsTable,
				idColumn:  audioIDColumn,
			},
			fkColumn:     tagIDColumn,
			foreignTable: tagTable,
			orderBy:      "COALESCE(tags.sort_name, tags.name) ASC",
		},
	}
)

type AudioStore struct {
	blobJoinQueryBuilder

	tableMgr *table
	oDateManager
	viewDateManager

	repo *storeRepository
}

func NewAudioStore(r *storeRepository, blobStore *BlobStore) *AudioStore {
	return &AudioStore{
		blobJoinQueryBuilder: blobJoinQueryBuilder{
			blobStore: blobStore,
			joinTable: audioTable,
		},

		tableMgr:        audioTableMgr,
		viewDateManager: viewDateManager{audiosViewTableMgr},
		oDateManager:    oDateManager{audiosOTableMgr},
		repo:            r,
	}
}

func (qb *AudioStore) table() exp.IdentifierExpression {
	return qb.tableMgr.table
}

func (qb *AudioStore) selectDataset() *goqu.SelectDataset {
	table := qb.table()
	files := fileTableMgr.table
	folders := folderTableMgr.table
	checksum := fingerprintTableMgr.table.As("fingerprint_md5")
	af := goqu.T(audioFilesTable)

	return dialect.From(table).LeftJoin(
		af,
		goqu.On(
			af.Col(audioIDColumn).Eq(table.Col(idColumn)),
			af.Col("primary").Eq(true),
		),
	).LeftJoin(
		files,
		goqu.On(files.Col(idColumn).Eq(af.Col("file_id"))),
	).LeftJoin(
		folders,
		goqu.On(folders.Col(idColumn).Eq(files.Col("parent_folder_id"))),
	).LeftJoin(
		checksum,
		goqu.On(
			checksum.Col(fileIDColumn).Eq(af.Col("file_id")),
			checksum.Col("type").Eq(models.FingerprintTypeMD5),
		),
	).Select(
		table.All(),
		files.Col(idColumn).As("primary_file_id"),
		checksum.Col("fingerprint").As("primary_file_checksum"),
		folders.Col("path").As("primary_file_folder_path"),
		files.Col("basename").As("primary_file_basename"),
	)
}

func (qb *AudioStore) Create(ctx context.Context, newObject *models.Audio, fileIDs []models.FileID) error {
	var r audioRow
	r.fromAudio(*newObject)

	id, err := qb.tableMgr.insertID(ctx, r)
	if err != nil {
		return err
	}

	if len(fileIDs) > 0 {
		const firstPrimary = true
		if err := audioFilesTableMgr.insertJoins(ctx, id, firstPrimary, fileIDs); err != nil {
			return err
		}
	}

	if newObject.URLs.Loaded() {
		const startPos = 0
		if err := audiosURLsTableMgr.insertJoins(ctx, id, startPos, newObject.URLs.List()); err != nil {
			return err
		}
	}

	if newObject.TagIDs.Loaded() {
		if err := audioTagsTableMgr.insertJoins(ctx, id, newObject.TagIDs.List()); err != nil {
			return err
		}
	}
	if newObject.PerformerIDs.Loaded() {
		if err := audioPerformersTableMgr.insertJoins(ctx, id, newObject.PerformerIDs.List()); err != nil {
			return err
		}
	}

	newObject.ID = id
	return nil
}

func (qb *AudioStore) UpdatePartial(ctx context.Context, id int, partial models.AudioPartial) (*models.Audio, error) {
	r := audioRowRecord{
		updateRecord{
			Record: make(exp.Record),
		},
	}
	r.fromPartial(partial)

	if len(r.Record) > 0 {
		if err := qb.tableMgr.updateByID(ctx, id, r.Record); err != nil {
			return nil, err
		}
	}

	if partial.URLs != nil {
		if err := audiosURLsTableMgr.modifyJoins(ctx, id, partial.URLs.Values, partial.URLs.Mode); err != nil {
			return nil, err
		}
	}

	if partial.TagIDs != nil {
		if err := audioTagsTableMgr.modifyJoins(ctx, id, partial.TagIDs.IDs, partial.TagIDs.Mode); err != nil {
			return nil, err
		}
	}
	if partial.PerformerIDs != nil {
		if err := audioPerformersTableMgr.modifyJoins(ctx, id, partial.PerformerIDs.IDs, partial.PerformerIDs.Mode); err != nil {
			return nil, err
		}
	}

	if partial.PrimaryFileID != nil {
		if err := audioFilesTableMgr.setPrimary(ctx, id, *partial.PrimaryFileID); err != nil {
			return nil, err
		}
	}

	return qb.find(ctx, id)
}

func (qb *AudioStore) Update(ctx context.Context, updatedObject *models.Audio) error {
	var r audioRow
	r.fromAudio(*updatedObject)

	if err := qb.tableMgr.updateByID(ctx, updatedObject.ID, r); err != nil {
		return err
	}

	if updatedObject.URLs.Loaded() {
		if err := audiosURLsTableMgr.replaceJoins(ctx, updatedObject.ID, updatedObject.URLs.List()); err != nil {
			return err
		}
	}

	if updatedObject.TagIDs.Loaded() {
		if err := audioTagsTableMgr.replaceJoins(ctx, updatedObject.ID, updatedObject.TagIDs.List()); err != nil {
			return err
		}
	}
	if updatedObject.PerformerIDs.Loaded() {
		if err := audioPerformersTableMgr.replaceJoins(ctx, updatedObject.ID, updatedObject.PerformerIDs.List()); err != nil {
			return err
		}
	}

	return nil
}

func (qb *AudioStore) Destroy(ctx context.Context, id int) error {
	return qb.tableMgr.destroyExisting(ctx, []int{id})
}

func (qb *AudioStore) Find(ctx context.Context, id int) (*models.Audio, error) {
	ret, err := qb.find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ret, err
}

func (qb *AudioStore) FindMany(ctx context.Context, ids []int) ([]*models.Audio, error) {
	q := qb.selectDataset().Where(qb.tableMgr.table.Col(idColumn).In(ids))
	unsorted, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.Audio, len(ids))

	for _, s := range unsorted {
		i := slices.Index(ids, s.ID)
		ret[i] = s
	}

	for i := range ret {
		if ret[i] == nil {
			return nil, fmt.Errorf("audio with id %d not found", ids[i])
		}
	}

	return ret, nil
}

func (qb *AudioStore) find(ctx context.Context, id int) (*models.Audio, error) {
	q := qb.selectDataset().Where(qb.tableMgr.byID(id))

	ret, err := qb.get(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("getting audio by id %d: %w", id, err)
	}

	return ret, nil
}

func (qb *AudioStore) findBySubquery(ctx context.Context, sq *goqu.SelectDataset) ([]*models.Audio, error) {
	table := qb.tableMgr.table

	q := qb.selectDataset().Where(
		table.Col(idColumn).In(sq),
	)

	return qb.getMany(ctx, q)
}

func (qb *AudioStore) get(ctx context.Context, q *goqu.SelectDataset) (*models.Audio, error) {
	ret, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, sql.ErrNoRows
	}

	return ret[0], nil
}

func (qb *AudioStore) getMany(ctx context.Context, q *goqu.SelectDataset) ([]*models.Audio, error) {
	const single = false
	var ret []*models.Audio
	if err := queryFunc(ctx, q, single, func(rows *sqlx.Rows) error {
		var f audioQueryRow
		if err := rows.StructScan(&f); err != nil {
			return fmt.Errorf("cannot scan audio row: %v", err)
		}
		ret = append(ret, f.resolve())
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *AudioStore) GetFiles(ctx context.Context, id int) ([]models.File, error) {
	fileIDs, err := audioFilesTableMgr.get(ctx, id)
	if err != nil {
		return nil, err
	}

	files, err := qb.repo.File.Find(ctx, fileIDs...)
	if err != nil {
		return nil, err
	}

	ret := make([]models.File, len(fileIDs))
	for i, fileID := range fileIDs {
		for _, f := range files {
			if f.Base().ID == fileID {
				ret[i] = f
				break
			}
		}
	}

	return ret, nil
}

func (qb *AudioStore) GetManyFileIDs(ctx context.Context, ids []int) ([][]models.FileID, error) {
	const primaryOnly = false
	return audioRepository.files.getMany(ctx, ids, primaryOnly)
}

func (qb *AudioStore) FindByFileID(ctx context.Context, fileID models.FileID) ([]*models.Audio, error) {
	audioFilesJoinTable := audioFilesJoinTable
	sq := dialect.From(audioFilesJoinTable).Select(audioFilesJoinTable.Col(audioIDColumn)).Where(audioFilesJoinTable.Col("file_id").Eq(fileID))
	ret, err := qb.findBySubquery(ctx, sq)

	if err != nil {
		return nil, fmt.Errorf("getting audio by file id %d: %w", fileID, err)
	}

	return ret, nil
}

func (qb *AudioStore) FindByFingerprints(ctx context.Context, fp []models.Fingerprint) ([]*models.Audio, error) {
	fingerprintTable := fingerprintTableMgr.table

	var ex []exp.Expression

	for _, v := range fp {
		ex = append(ex, goqu.And(
			fingerprintTable.Col("type").Eq(v.Type),
			fingerprintTable.Col("fingerprint").Eq(v.Fingerprint),
		))
	}

	sq := dialect.From(audioFilesJoinTable).
		InnerJoin(
			fingerprintTable,
			goqu.On(fingerprintTable.Col(fileIDColumn).Eq(audioFilesJoinTable.Col(fileIDColumn))),
		).
		Select(audioFilesJoinTable.Col(audioIDColumn)).Where(goqu.Or(ex...))

	ret, err := qb.findBySubquery(ctx, sq)
	if err != nil {
		return nil, fmt.Errorf("getting audio by fingerprints: %w", err)
	}

	return ret, nil
}

func (qb *AudioStore) FindByPrimaryFileID(ctx context.Context, fileID models.FileID) ([]*models.Audio, error) {
	audioFilesJoinTable := audioFilesJoinTable
	sq := dialect.From(audioFilesJoinTable).Select(audioFilesJoinTable.Col(audioIDColumn)).Where(
		audioFilesJoinTable.Col("file_id").Eq(fileID),
		audioFilesJoinTable.Col("primary").Eq(true),
	)
	ret, err := qb.findBySubquery(ctx, sq)

	if err != nil {
		return nil, fmt.Errorf("getting audio by primary file id %d: %w", fileID, err)
	}

	return ret, nil
}

func (qb *AudioStore) CountByFileID(ctx context.Context, fileID models.FileID) (int, error) {
	joinTable := goqu.T(audioFilesTable)
	q := dialect.Select(goqu.COUNT("*")).From(joinTable).Where(joinTable.Col("file_id").Eq(fileID))
	return count(ctx, q)
}

func (qb *AudioStore) FindByChecksum(ctx context.Context, checksum string) ([]*models.Audio, error) {
	table := qb.table()
	files := goqu.T("files")
	audioFiles := goqu.T(audioFilesTable)
	fingerprints := fingerprintTableMgr.table

	sq := dialect.From(table).
		InnerJoin(audioFiles, goqu.On(audioFiles.Col(audioIDColumn).Eq(table.Col(idColumn)))).
		InnerJoin(files, goqu.On(files.Col(idColumn).Eq(audioFiles.Col("file_id")))).
		InnerJoin(fingerprints, goqu.On(
			fingerprints.Col(fileIDColumn).Eq(files.Col(idColumn)),
			fingerprints.Col("type").Eq(models.FingerprintTypeMD5),
		)).
		Select(table.Col(idColumn)).
		Where(fingerprints.Col("fingerprint").Eq(checksum))

	ret, err := qb.findBySubquery(ctx, sq)
	if err != nil {
		return nil, fmt.Errorf("getting audio by checksum %s: %w", checksum, err)
	}

	return ret, nil
}

func (qb *AudioStore) FindByPath(ctx context.Context, p string) (*models.Audio, error) {
	table := qb.table()
	files := goqu.T("files")
	folders := goqu.T("folders")
	audioFiles := goqu.T(audioFilesTable)

	basename := filepath.Base(p)
	dir := filepath.Dir(p)

	sq := dialect.From(table).
		InnerJoin(audioFiles, goqu.On(audioFiles.Col(audioIDColumn).Eq(table.Col(idColumn)))).
		InnerJoin(files, goqu.On(files.Col(idColumn).Eq(audioFiles.Col("file_id")))).
		InnerJoin(folders, goqu.On(folders.Col(idColumn).Eq(files.Col("parent_folder_id")))).
		Select(table.Col(idColumn)).
		Where(
			files.Col("basename").Eq(basename),
			folders.Col("path").Eq(dir),
		)

	ret, err := qb.findBySubquery(ctx, sq)
	if err != nil {
		return nil, fmt.Errorf("getting audio by path %s: %w", p, err)
	}

	if len(ret) == 0 {
		return nil, nil
	}

	return ret[0], nil
}

func (qb *AudioStore) FindByPerformerID(ctx context.Context, performerID int) ([]*models.Audio, error) {
	audioPerformersJoinTable := audioPerformersJoinTable
	sq := dialect.From(audioPerformersJoinTable).Select(audioPerformersJoinTable.Col(audioIDColumn)).Where(
		audioPerformersJoinTable.Col(performerIDColumn).Eq(performerID),
	)

	ret, err := qb.findBySubquery(ctx, sq)
	if err != nil {
		return nil, fmt.Errorf("getting audio by performer %d: %w", performerID, err)
	}

	return ret, nil
}

func (qb *AudioStore) Count(ctx context.Context) (int, error) {
	q := dialect.Select(goqu.COUNT("*")).From(qb.table())
	return count(ctx, q)
}

func (qb *AudioStore) CountByPerformerID(ctx context.Context, performerID int) (int, error) {
	audioPerformersJoinTable := audioPerformersJoinTable
	q := dialect.Select(goqu.COUNT("*")).From(audioPerformersJoinTable).Where(audioPerformersJoinTable.Col(performerIDColumn).Eq(performerID))
	return count(ctx, q)
}

func (qb *AudioStore) OCountByPerformerID(ctx context.Context, performerID int) (int, error) {
	table := qb.table()
	joinTable := audioPerformersJoinTable
	oHistoryTable := goqu.T(audiosODatesTable)

	q := dialect.Select(goqu.COUNT("*")).From(table).InnerJoin(
		oHistoryTable,
		goqu.On(table.Col(idColumn).Eq(oHistoryTable.Col(audioIDColumn))),
	).InnerJoin(
		joinTable,
		goqu.On(
			table.Col(idColumn).Eq(joinTable.Col(audioIDColumn)),
		),
	).Where(joinTable.Col(performerIDColumn).Eq(performerID))

	var ret int
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

func (qb *AudioStore) OCount(ctx context.Context) (int, error) {
	oHistoryTable := goqu.T(audiosODatesTable)

	q := dialect.Select(goqu.COUNT("*")).From(oHistoryTable)
	var ret int
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

func (qb *AudioStore) Size(ctx context.Context) (float64, error) {
	table := qb.table()
	files := goqu.T("files")
	audioFiles := goqu.T(audioFilesTable)

	q := dialect.From(table).
		InnerJoin(audioFiles, goqu.On(audioFiles.Col(audioIDColumn).Eq(table.Col(idColumn)))).
		InnerJoin(files, goqu.On(files.Col(idColumn).Eq(audioFiles.Col("file_id")))).
		Select(goqu.SUM("files.size"))

	var ret sql.NullFloat64
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret.Float64, nil
}

func (qb *AudioStore) Duration(ctx context.Context) (float64, error) {
	table := qb.table()
	audioFiles := goqu.T(audioFilesTable)
	files := goqu.T("files")
	audioFilesTbl := goqu.T("audio_files")

	q := dialect.From(table).
		InnerJoin(audioFiles, goqu.On(audioFiles.Col(audioIDColumn).Eq(table.Col(idColumn)))).
		InnerJoin(files, goqu.On(files.Col(idColumn).Eq(audioFiles.Col("file_id")))).
		InnerJoin(audioFilesTbl, goqu.On(audioFilesTbl.Col("file_id").Eq(files.Col(idColumn)))).
		Select(goqu.SUM("audio_files.duration"))

	var ret sql.NullFloat64
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret.Float64, nil
}

func (qb *AudioStore) All(ctx context.Context) ([]*models.Audio, error) {
	table := qb.tableMgr.table
	return qb.getMany(ctx, qb.selectDataset().Order(table.Col("title").Asc()))
}

func (qb *AudioStore) makeQuery(ctx context.Context, audioFilter *models.AudioFilterType, findFilter *models.FindFilterType) (*queryBuilder, error) {
	if audioFilter == nil {
		audioFilter = &models.AudioFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := audioRepository.newQuery()
	distinctIDs(&query, audioTable)

	if q := findFilter.Q; q != nil && *q != "" {
		query.addJoins(
			join{
				table:    audioFilesTable,
				onClause: "audios_files.audio_id = audios.id",
			},
			join{
				table:    fileTable,
				onClause: "audios_files.file_id = files.id",
			},
			join{
				table:    folderTable,
				onClause: "files.parent_folder_id = folders.id",
			},
			join{
				table:    fingerprintTable,
				onClause: "files_fingerprints.file_id = audios_files.file_id",
			},
		)

		filepathColumn := "folders.path || '" + string(filepath.Separator) + "' || files.basename"
		searchColumns := []string{"audios.title", "audios.details", filepathColumn, "files_fingerprints.fingerprint"}
		query.parseQueryString(searchColumns, *q)
	}

	filter := filterBuilderFromHandler(ctx, &audioFilterHandler{
		audioFilter: audioFilter,
	})

	if err := query.addFilter(filter); err != nil {
		return nil, err
	}

	if err := qb.setAudioSort(&query, findFilter); err != nil {
		return nil, err
	}
	query.sortAndPagination += getPagination(findFilter)

	return &query, nil
}

func (qb *AudioStore) Query(ctx context.Context, options models.AudioQueryOptions) (*models.AudioQueryResult, error) {
	query, err := qb.makeQuery(ctx, options.AudioFilter, options.FindFilter)
	if err != nil {
		return nil, err
	}

	result := models.NewAudioQueryResult(qb)

	if options.TotalDuration {
		result.TotalDuration, err = qb.Duration(ctx)
		if err != nil {
			return nil, err
		}
	}
	if options.TotalSize {
		result.TotalSize, err = qb.Size(ctx)
		if err != nil {
			return nil, err
		}
	}

	idsResult, countResult, err := query.executeFind(ctx)
	if err != nil {
		return nil, err
	}

	result.IDs = idsResult
	result.Count = countResult

	return result, nil
}

func (qb *AudioStore) QueryCount(ctx context.Context, audioFilter *models.AudioFilterType, findFilter *models.FindFilterType) (int, error) {
	query, err := qb.makeQuery(ctx, audioFilter, findFilter)
	if err != nil {
		return 0, err
	}

	return query.executeCount(ctx)
}

var audioSortOptions = sortOptions{
	"bitrate",
	"channels",
	"created_at",
	"date",
	"duration",
	"file_count",
	"filesize",
	"file_mod_time",
	"id",
	"last_o_at",
	"last_played_at",
	"o_counter",
	"organized",
	"path",
	"performer_age",
	"performer_count",
	"play_count",
	"play_duration",
	"random",
	"rating",
	"resume_time",
	"sample_rate",
	"tag_count",
	"title",
	"updated_at",
}

func (qb *AudioStore) setAudioSort(query *queryBuilder, findFilter *models.FindFilterType) error {
	if findFilter == nil || findFilter.Sort == nil || *findFilter.Sort == "" {
		return nil
	}
	sort := findFilter.GetSort("title")

	// CVE-2024-32231 - ensure sort is in the list of allowed sorts
	if err := audioSortOptions.validateSort(sort); err != nil {
		return err
	}

	addFilesJoin := func() {
		query.addJoins(
			join{
				table:    audioFilesTable,
				onClause: "audios_files.audio_id = audios.id",
			},
			join{
				table:    fileTable,
				onClause: "audios_files.file_id = files.id",
			},
		)
	}

	addAudioFilesJoin := func() {
		addFilesJoin()
		query.addJoins(
			join{
				table:    audioFileTable,
				onClause: "audio_files.file_id = audios_files.file_id",
			},
		)
	}

	addFolderJoin := func() {
		query.addJoins(
			join{
				table:    folderTable,
				onClause: "files.parent_folder_id = folders.id",
			},
		)
	}

	direction := findFilter.GetDirection()
	switch sort {
	case "tag_count":
		query.sortAndPagination += getCountSort(audioTable, audioTagsTable, audioIDColumn, direction)
	case "performer_count":
		query.sortAndPagination += getCountSort(audioTable, audioPerformersTable, audioIDColumn, direction)
	case "file_count":
		query.sortAndPagination += getCountSort(audioTable, audioFilesTable, audioIDColumn, direction)
	case "path":
		// special handling for path
		addFilesJoin()
		addFolderJoin()
		query.sortAndPagination += fmt.Sprintf(" ORDER BY COALESCE(folders.path, '') || COALESCE(files.basename, '') COLLATE NATURAL_CI %s", getSortDirection(direction))
	case "bitrate":
		addAudioFilesJoin()
		query.sortAndPagination += getSort(sort, direction, audioFileTable)
	case "channels":
		addAudioFilesJoin()
		query.sortAndPagination += getSort(sort, direction, audioFileTable)
	case "duration":
		addAudioFilesJoin()
		query.sortAndPagination += getSort(sort, direction, audioFileTable)
	case "sample_rate":
		addAudioFilesJoin()
		query.sortAndPagination += getSort(sort, direction, audioFileTable)
	case "file_mod_time":
		sort = "mod_time"
		addFilesJoin()
		query.sortAndPagination += getSort(sort, direction, fileTable)
	case "filesize":
		sort = "size"
		addFilesJoin()
		query.sortAndPagination += getSort(sort, direction, fileTable)
	case "title":
		addFilesJoin()
		addFolderJoin()
		query.sortAndPagination += " ORDER BY COALESCE(audios.title, files.basename) COLLATE NATURAL_CI " + getSortDirection(direction) + ", folders.path COLLATE NATURAL_CI " + getSortDirection(direction)
	case "play_count":
		query.sortAndPagination += getCountSort(audioTable, audiosViewDatesTable, audioIDColumn, direction)
	case "last_played_at":
		query.sortAndPagination += fmt.Sprintf(" ORDER BY (SELECT MAX(view_date) FROM %s AS sort WHERE sort.%s = %s.id) %s", audiosViewDatesTable, audioIDColumn, audioTable, getSortDirection(direction))
	case "last_o_at":
		query.sortAndPagination += fmt.Sprintf(" ORDER BY (SELECT MAX(o_date) FROM %s AS sort WHERE sort.%s = %s.id) %s", audiosODatesTable, audioIDColumn, audioTable, getSortDirection(direction))
	case "o_counter":
		query.sortAndPagination += getCountSort(audioTable, audiosODatesTable, audioIDColumn, direction)
	case "performer_age":
		aggregation := "MIN"
		if direction == "DESC" {
			aggregation = "MAX"
		}
		fallback := "NULL"
		if direction == "ASC" {
			fallback = "9223372036854775807"
		}
		query.sortAndPagination += fmt.Sprintf(
			" ORDER BY (SELECT COALESCE(%s(JulianDay(audios.date) - JulianDay(performers.birthdate)), %s) FROM %s as performers INNER JOIN %s AS aggregation WHERE performers.id = aggregation.%s AND aggregation.%s = %s.id) %s",
			aggregation,
			fallback,
			performerTable,
			audioPerformersTable,
			performerIDColumn,
			audioIDColumn,
			audioTable,
			getSortDirection(direction),
		)
	default:
		query.sortAndPagination += getSort(sort, direction, "audios")
	}

	// Whatever the sorting, always use title/id as a final sort
	query.sortAndPagination += ", COALESCE(audios.title, audios.id) COLLATE NATURAL_CI ASC"

	return nil
}

func (qb *AudioStore) AddFileID(ctx context.Context, id int, fileID models.FileID) error {
	const primary = false
	return audioFilesTableMgr.insertJoin(ctx, id, primary, fileID)
}

func (qb *AudioStore) AssignFiles(ctx context.Context, audioID int, fileIDs []models.FileID) error {
	return audioFilesTableMgr.replaceJoins(ctx, audioID, fileIDs)
}

func (qb *AudioStore) GetPerformerIDs(ctx context.Context, id int) ([]int, error) {
	return audioRepository.performers.getIDs(ctx, id)
}

func (qb *AudioStore) GetTagIDs(ctx context.Context, id int) ([]int, error) {
	return audioRepository.tags.getIDs(ctx, id)
}

func (qb *AudioStore) GetURLs(ctx context.Context, audioID int) ([]string, error) {
	return audiosURLsTableMgr.get(ctx, audioID)
}

func (qb *AudioStore) SaveActivity(ctx context.Context, id int, resumeTime *float64, playDuration *float64) (bool, error) {
	if err := qb.tableMgr.checkIDExists(ctx, id); err != nil {
		return false, err
	}

	record := goqu.Record{}

	if resumeTime != nil {
		record["resume_time"] = *resumeTime
	}

	if playDuration != nil {
		record["play_duration"] = goqu.L("play_duration + ?", playDuration)
	}

	if len(record) > 0 {
		if err := qb.tableMgr.updateByID(ctx, id, record); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (qb *AudioStore) ResetActivity(ctx context.Context, id int, resetResume bool, resetDuration bool) (bool, error) {
	if err := qb.tableMgr.checkIDExists(ctx, id); err != nil {
		return false, err
	}

	record := goqu.Record{}

	if resetResume {
		record["resume_time"] = 0.0
	}

	if resetDuration {
		record["play_duration"] = 0.0
	}

	if len(record) > 0 {
		if err := qb.tableMgr.updateByID(ctx, id, record); err != nil {
			return false, err
		}
	}

	return true, nil
}

// DeleteViews overrides the embedded viewDateManager.DeleteViews to validate audio ID exists
func (qb *AudioStore) DeleteViews(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	// Validate that the audio ID exists before attempting to delete views
	if err := qb.tableMgr.checkIDExists(ctx, id); err != nil {
		return nil, err
	}

	// Delegate to the embedded viewDateManager
	return qb.viewDateManager.DeleteViews(ctx, id, dates)
}

// AddViews overrides the embedded viewDateManager.AddViews to validate audio ID exists
func (qb *AudioStore) AddViews(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	// Validate that the audio ID exists before attempting to add views
	if err := qb.tableMgr.checkIDExists(ctx, id); err != nil {
		return nil, err
	}

	// Delegate to the embedded viewDateManager
	return qb.viewDateManager.AddViews(ctx, id, dates)
}

// Cover image methods
func (qb *AudioStore) GetCover(ctx context.Context, audioID int) ([]byte, error) {
	return qb.GetImage(ctx, audioID, audioCoverBlobColumn)
}

func (qb *AudioStore) HasCover(ctx context.Context, audioID int) (bool, error) {
	return qb.HasImage(ctx, audioID, audioCoverBlobColumn)
}

func (qb *AudioStore) UpdateCover(ctx context.Context, audioID int, image []byte) error {
	return qb.UpdateImage(ctx, audioID, audioCoverBlobColumn, image)
}
