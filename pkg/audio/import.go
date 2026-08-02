package audio

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/jsonschema"
)

type ImporterReaderWriter interface {
	models.AudioCreatorUpdater
	FindByFileID(ctx context.Context, fileID models.FileID) ([]*models.Audio, error)
}

type Importer struct {
	ReaderWriter        ImporterReaderWriter
	FileFinder          models.FileFinder
	StudioWriter        models.StudioFinderCreator
	PerformerWriter     models.PerformerFinderCreator
	TagWriter           models.TagFinderCreator
	Input               jsonschema.Audio
	MissingRefBehaviour models.ImportMissingRefEnum

	ID    int
	audio models.Audio
}

func (i *Importer) PreImport(ctx context.Context) error {
	i.audio = i.audioJSONToAudio(i.Input)

	if err := i.populateFiles(ctx); err != nil {
		return err
	}

	if err := i.populatePerformers(ctx); err != nil {
		return err
	}

	if err := i.populateTags(ctx); err != nil {
		return err
	}

	return nil
}

func (i *Importer) audioJSONToAudio(audioJSON jsonschema.Audio) models.Audio {
	newAudio := models.Audio{
		PerformerIDs: models.NewRelatedIDs([]int{}),
		TagIDs:       models.NewRelatedIDs([]int{}),

		Title:     audioJSON.Title,
		Details:   audioJSON.Details,
		Organized: audioJSON.Organized,
		CreatedAt: audioJSON.CreatedAt.GetTime(),
		UpdatedAt: audioJSON.UpdatedAt.GetTime(),
	}

	// Convert single URL to URLs list
	if audioJSON.URL != "" {
		newAudio.URLs = models.NewRelatedStrings([]string{audioJSON.URL})
	} else {
		newAudio.URLs = models.NewRelatedStrings([]string{})
	}

	if audioJSON.Rating != 0 {
		newAudio.Rating = &audioJSON.Rating
	}

	if audioJSON.Date != "" {
		d, err := models.ParseDate(audioJSON.Date)
		if err == nil {
			newAudio.Date = &d
		}
	}

	return newAudio
}

func (i *Importer) populateFiles(ctx context.Context) error {
	files := make([]models.File, 0)

	for _, ref := range i.Input.Files {
		path := ref
		f, err := i.FileFinder.FindByPath(ctx, path, true)
		if err != nil {
			return fmt.Errorf("error finding file: %w", err)
		}

		if f == nil {
			return fmt.Errorf("audio file '%s' not found", path)
		} else {
			files = append(files, f)
		}
	}

	i.audio.Files = models.NewRelatedFiles(files)

	return nil
}

func (i *Importer) populatePerformers(ctx context.Context) error {
	if len(i.Input.Performers) > 0 {
		performers, err := i.PerformerWriter.FindByNames(ctx, i.Input.Performers, false)
		if err != nil {
			return fmt.Errorf("error finding performers: %w", err)
		}

		var pluckedNames []string
		for _, p := range performers {
			pluckedNames = append(pluckedNames, p.Name)
		}

		missingPerformers := i.getMissingNames(i.Input.Performers, pluckedNames)
		if len(missingPerformers) > 0 {
			if i.MissingRefBehaviour == models.ImportMissingRefEnumFail {
				return fmt.Errorf("audio performers [%v] not found", missingPerformers)
			}

			if i.MissingRefBehaviour == models.ImportMissingRefEnumCreate {
				createdPerformers, err := i.createPerformers(ctx, missingPerformers)
				if err != nil {
					return fmt.Errorf("error creating performers: %w", err)
				}

				performers = append(performers, createdPerformers...)
			}

			// models.ImportMissingRefEnumIgnore needs no handling
		}

		var performerIDs []int
		for _, p := range performers {
			performerIDs = append(performerIDs, p.ID)
		}
		i.audio.PerformerIDs = models.NewRelatedIDs(performerIDs)
	}

	return nil
}

func (i *Importer) createPerformers(ctx context.Context, names []string) ([]*models.Performer, error) {
	var ret []*models.Performer
	for _, name := range names {
		newPerformer := models.CreatePerformerInput{
			Performer: &models.Performer{
				Name: name,
			},
		}

		err := i.PerformerWriter.Create(ctx, &newPerformer)
		if err != nil {
			return nil, err
		}

		performer := &models.Performer{
			ID:   newPerformer.Performer.ID,
			Name: name,
		}
		ret = append(ret, performer)
	}

	return ret, nil
}

func (i *Importer) populateTags(ctx context.Context) error {
	if len(i.Input.Tags) > 0 {
		tags, err := importTags(ctx, i.TagWriter, i.Input.Tags, i.MissingRefBehaviour)
		if err != nil {
			return err
		}

		var tagIDs []int
		for _, t := range tags {
			tagIDs = append(tagIDs, t.ID)
		}
		i.audio.TagIDs = models.NewRelatedIDs(tagIDs)
	}

	return nil
}

func (i *Importer) getMissingNames(expected []string, actual []string) []string {
	var missing []string
	for _, expectedName := range expected {
		found := false
		for _, actualName := range actual {
			if expectedName == actualName {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, expectedName)
		}
	}
	return missing
}

func (i *Importer) PostImport(ctx context.Context, id int) error {
	return nil
}

func (i *Importer) Name() string {
	if i.Input.Title != "" {
		return i.Input.Title
	}

	if len(i.Input.Files) > 0 {
		return i.Input.Files[0]
	}

	return fmt.Sprintf("<audio %d>", i.ID)
}

func (i *Importer) FindExistingID(ctx context.Context) (*int, error) {
	if len(i.audio.Files.List()) == 0 {
		return nil, nil
	}

	existing, err := i.ReaderWriter.FindByFileID(ctx, i.audio.Files.List()[0].Base().ID)
	if err != nil {
		return nil, err
	}

	if len(existing) > 0 {
		id := existing[0].ID
		return &id, nil
	}

	return nil, nil
}

func (i *Importer) Create(ctx context.Context) (*int, error) {
	var fileIDs []models.FileID
	for _, file := range i.audio.Files.List() {
		fileIDs = append(fileIDs, file.Base().ID)
	}

	// Ensure we always pass a non-nil slice
	if fileIDs == nil {
		fileIDs = []models.FileID{}
	}

	err := i.ReaderWriter.Create(ctx, &i.audio, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("error creating audio: %w", err)
	}

	id := i.audio.ID
	return &id, nil
}

func (i *Importer) Update(ctx context.Context, id int) error {
	partial := models.NewAudioPartial()
	i.audio.ID = id

	_, err := i.ReaderWriter.UpdatePartial(ctx, id, partial)
	if err != nil {
		return fmt.Errorf("error updating audio: %w", err)
	}

	return nil
}

func importTags(ctx context.Context, tagWriter models.TagFinderCreator, names []string, missingRefBehaviour models.ImportMissingRefEnum) ([]*models.Tag, error) {
	tags, err := tagWriter.FindByNames(ctx, names, false)
	if err != nil {
		return nil, err
	}

	var pluckedNames []string
	for _, t := range tags {
		pluckedNames = append(pluckedNames, t.Name)
	}

	var missingTags []string
	for _, name := range names {
		found := false
		for _, pluckedName := range pluckedNames {
			if name == pluckedName {
				found = true
				break
			}
		}
		if !found {
			missingTags = append(missingTags, name)
		}
	}

	if len(missingTags) > 0 {
		if missingRefBehaviour == models.ImportMissingRefEnumFail {
			return nil, fmt.Errorf("tags [%v] not found", missingTags)
		}

		if missingRefBehaviour == models.ImportMissingRefEnumCreate {
			createdTags, err := createTags(ctx, tagWriter, missingTags)
			if err != nil {
				return nil, fmt.Errorf("error creating tags: %w", err)
			}

			tags = append(tags, createdTags...)
		}

		// models.ImportMissingRefEnumIgnore needs no handling
	}

	return tags, nil
}

func createTags(ctx context.Context, tagWriter models.TagCreator, names []string) ([]*models.Tag, error) {
	var ret []*models.Tag
	for _, name := range names {
		newTag := models.NewTag()
		newTag.Name = name

		err := tagWriter.Create(ctx, &models.CreateTagInput{
			Tag: &newTag,
		})
		if err != nil {
			return nil, err
		}

		ret = append(ret, &newTag)
	}

	return ret, nil
}
