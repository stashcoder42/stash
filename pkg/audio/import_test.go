package audio

import (
	"errors"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/jsonschema"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	existingPerformerID = 103
	existingTagID       = 105
	existingFileID      = models.FileID(107)

	existingPerformerName = "existingPerformerName"
	existingPerformerErr  = "existingPerformerErr"
	missingPerformerName  = "missingPerformerName"

	existingTagName = "existingTagName"
	existingTagErr  = "existingTagErr"
	missingTagName  = "missingTagName"

	existingFilePath = "/path/to/audio.mp3"
	missingFilePath  = "/path/to/missing.mp3"
)

func TestImporterPreImport(t *testing.T) {
	i := Importer{}

	err := i.PreImport(testCtx)
	assert.Nil(t, err)
}

func TestImporterPreImportWithFile(t *testing.T) {
	// Create a simple test file that implements the File interface
	testFile := &models.BaseFile{
		ID:   existingFileID,
		Path: existingFilePath,
	}

	db := mocks.NewDatabase()
	db.File.On("FindByPath", testCtx, existingFilePath, true).Return(testFile, nil).Once()

	i := Importer{
		FileFinder: db.File,
		Input: jsonschema.Audio{
			Files: []string{existingFilePath},
		},
	}

	err := i.PreImport(testCtx)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(i.audio.Files.List()))

	db.AssertExpectations(t)
}

func TestImporterPreImportWithMissingFile(t *testing.T) {
	db := mocks.NewDatabase()
	db.File.On("FindByPath", testCtx, missingFilePath, true).Return(nil, nil).Once()

	i := Importer{
		FileFinder: db.File,
		Input: jsonschema.Audio{
			Files: []string{missingFilePath},
		},
	}

	err := i.PreImport(testCtx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")

	db.AssertExpectations(t)
}

func TestImporterPreImportWithPerformer(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		PerformerWriter:     db.Performer,
		MissingRefBehaviour: models.ImportMissingRefEnumFail,
		Input: jsonschema.Audio{
			Performers: []string{
				existingPerformerName,
			},
		},
	}

	db.Performer.On("FindByNames", testCtx, []string{existingPerformerName}, false).Return([]*models.Performer{
		{
			ID:   existingPerformerID,
			Name: existingPerformerName,
		},
	}, nil).Once()
	db.Performer.On("FindByNames", testCtx, []string{existingPerformerErr}, false).Return(nil, errors.New("FindByNames error")).Once()

	err := i.PreImport(testCtx)
	assert.Nil(t, err)
	assert.Equal(t, []int{existingPerformerID}, i.audio.PerformerIDs.List())

	i.Input.Performers = []string{existingPerformerErr}
	err = i.PreImport(testCtx)
	assert.NotNil(t, err)

	db.AssertExpectations(t)
}

func TestImporterPreImportWithMissingPerformer(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		PerformerWriter: db.Performer,
		Input: jsonschema.Audio{
			Performers: []string{
				missingPerformerName,
			},
		},
		MissingRefBehaviour: models.ImportMissingRefEnumFail,
	}

	db.Performer.On("FindByNames", testCtx, []string{missingPerformerName}, false).Return(nil, nil).Times(3)
	db.Performer.On("Create", testCtx, mock.AnythingOfType("*models.CreatePerformerInput")).Run(func(args mock.Arguments) {
		performer := args.Get(1).(*models.CreatePerformerInput)
		performer.Performer.ID = existingPerformerID
	}).Return(nil)

	err := i.PreImport(testCtx)
	assert.NotNil(t, err)

	i.MissingRefBehaviour = models.ImportMissingRefEnumIgnore
	err = i.PreImport(testCtx)
	assert.Nil(t, err)

	i.MissingRefBehaviour = models.ImportMissingRefEnumCreate
	err = i.PreImport(testCtx)
	assert.Nil(t, err)
	assert.Equal(t, []int{existingPerformerID}, i.audio.PerformerIDs.List())

	db.AssertExpectations(t)
}

func TestImporterPreImportWithMissingPerformerCreateErr(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		PerformerWriter: db.Performer,
		Input: jsonschema.Audio{
			Performers: []string{
				missingPerformerName,
			},
		},
		MissingRefBehaviour: models.ImportMissingRefEnumCreate,
	}

	db.Performer.On("FindByNames", testCtx, []string{missingPerformerName}, false).Return(nil, nil).Once()
	db.Performer.On("Create", testCtx, mock.AnythingOfType("*models.CreatePerformerInput")).Return(errors.New("Create error"))

	err := i.PreImport(testCtx)
	assert.NotNil(t, err)

	db.AssertExpectations(t)
}

func TestImporterPreImportWithTag(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		TagWriter:           db.Tag,
		MissingRefBehaviour: models.ImportMissingRefEnumFail,
		Input: jsonschema.Audio{
			Tags: []string{
				existingTagName,
			},
		},
	}

	db.Tag.On("FindByNames", testCtx, []string{existingTagName}, false).Return([]*models.Tag{
		{
			ID:   existingTagID,
			Name: existingTagName,
		},
	}, nil).Once()
	db.Tag.On("FindByNames", testCtx, []string{existingTagErr}, false).Return(nil, errors.New("FindByNames error")).Once()

	err := i.PreImport(testCtx)
	assert.Nil(t, err)
	assert.Equal(t, []int{existingTagID}, i.audio.TagIDs.List())

	i.Input.Tags = []string{existingTagErr}
	err = i.PreImport(testCtx)
	assert.NotNil(t, err)

	db.AssertExpectations(t)
}

func TestImporterPreImportWithMissingTag(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		TagWriter: db.Tag,
		Input: jsonschema.Audio{
			Tags: []string{
				missingTagName,
			},
		},
		MissingRefBehaviour: models.ImportMissingRefEnumFail,
	}

	db.Tag.On("FindByNames", testCtx, []string{missingTagName}, false).Return(nil, nil).Times(3)
	db.Tag.On("Create", testCtx, mock.AnythingOfType("*models.CreateTagInput")).Run(func(args mock.Arguments) {
		tag := args.Get(1).(*models.CreateTagInput)
		tag.Tag.ID = existingTagID
	}).Return(nil)

	err := i.PreImport(testCtx)
	assert.NotNil(t, err)

	i.MissingRefBehaviour = models.ImportMissingRefEnumIgnore
	err = i.PreImport(testCtx)
	assert.Nil(t, err)

	i.MissingRefBehaviour = models.ImportMissingRefEnumCreate
	err = i.PreImport(testCtx)
	assert.Nil(t, err)
	assert.Equal(t, []int{existingTagID}, i.audio.TagIDs.List())

	db.AssertExpectations(t)
}

func TestImporterPreImportWithMissingTagCreateErr(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		TagWriter: db.Tag,
		Input: jsonschema.Audio{
			Tags: []string{
				missingTagName,
			},
		},
		MissingRefBehaviour: models.ImportMissingRefEnumCreate,
	}

	db.Tag.On("FindByNames", testCtx, []string{missingTagName}, false).Return(nil, nil).Once()
	db.Tag.On("Create", testCtx, mock.AnythingOfType("*models.CreateTagInput")).Return(errors.New("Create error"))

	err := i.PreImport(testCtx)
	assert.NotNil(t, err)

	db.AssertExpectations(t)
}

func TestImporterName(t *testing.T) {
	i := Importer{
		Input: jsonschema.Audio{
			Title: "Test Audio",
		},
	}

	assert.Equal(t, "Test Audio", i.Name())

	i.Input.Title = ""
	i.Input.Files = []string{"/path/to/audio.mp3"}
	assert.Equal(t, "/path/to/audio.mp3", i.Name())

	i.Input.Files = []string{}
	i.ID = 123
	assert.Equal(t, "<audio 123>", i.Name())
}

func TestImporterFindExistingID(t *testing.T) {
	testFile := &models.BaseFile{
		ID:   existingFileID,
		Path: existingFilePath,
	}

	db := mocks.NewDatabase()

	i := Importer{
		ReaderWriter: db.Audio,
		audio: models.Audio{
			Files: models.NewRelatedFiles([]models.File{testFile}),
		},
	}

	existingAudio := &models.Audio{ID: 456}
	db.Audio.On("FindByFileID", testCtx, existingFileID).Return([]*models.Audio{existingAudio}, nil).Once()

	id, err := i.FindExistingID(testCtx)
	assert.Nil(t, err)
	assert.Equal(t, 456, *id)

	// Test no existing audio
	db.Audio.On("FindByFileID", testCtx, existingFileID).Return([]*models.Audio{}, nil).Once()
	id, err = i.FindExistingID(testCtx)
	assert.Nil(t, err)
	assert.Nil(t, id)

	// Test no files
	i.audio.Files = models.NewRelatedFiles([]models.File{})
	id, err = i.FindExistingID(testCtx)
	assert.Nil(t, err)
	assert.Nil(t, id)

	db.AssertExpectations(t)
}

func TestImporterCreate(t *testing.T) {
	testFile := &models.BaseFile{
		ID:   existingFileID,
		Path: existingFilePath,
	}

	db := mocks.NewDatabase()

	i := Importer{
		ReaderWriter: db.Audio,
		audio: models.Audio{
			Title: "Test Audio",
			Files: models.NewRelatedFiles([]models.File{testFile}),
		},
	}

	db.Audio.On("Create", testCtx, &i.audio, []models.FileID{existingFileID}).Run(func(args mock.Arguments) {
		audio := args.Get(1).(*models.Audio)
		audio.ID = 789
	}).Return(nil).Once()

	id, err := i.Create(testCtx)
	assert.Nil(t, err)
	assert.Equal(t, 789, *id)

	db.AssertExpectations(t)
}

func TestImporterCreateError(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		ReaderWriter: db.Audio,
		audio: models.Audio{
			Title: "Test Audio",
			Files: models.NewRelatedFiles([]models.File{}),
		},
	}

	db.Audio.On("Create", testCtx, &i.audio, []models.FileID{}).Return(errors.New("Create error")).Once()

	id, err := i.Create(testCtx)
	assert.NotNil(t, err)
	assert.Nil(t, id)

	db.AssertExpectations(t)
}

func TestImporterUpdate(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		ReaderWriter: db.Audio,
		audio: models.Audio{
			Title: "Test Audio",
		},
	}

	updatedAudio := &models.Audio{ID: 123, Title: "Updated Audio"}
	db.Audio.On("UpdatePartial", testCtx, 123, mock.AnythingOfType("models.AudioPartial")).Return(updatedAudio, nil).Once()

	err := i.Update(testCtx, 123)
	assert.Nil(t, err)
	assert.Equal(t, 123, i.audio.ID)

	db.AssertExpectations(t)
}

func TestImporterUpdateError(t *testing.T) {
	db := mocks.NewDatabase()

	i := Importer{
		ReaderWriter: db.Audio,
		audio: models.Audio{
			Title: "Test Audio",
		},
	}

	db.Audio.On("UpdatePartial", testCtx, 123, mock.AnythingOfType("models.AudioPartial")).Return(nil, errors.New("Update error")).Once()

	err := i.Update(testCtx, 123)
	assert.NotNil(t, err)

	db.AssertExpectations(t)
}

func TestImporterGetMissingNames(t *testing.T) {
	i := Importer{}

	expected := []string{"a", "b", "c"}
	actual := []string{"b", "d"}

	missing := i.getMissingNames(expected, actual)
	assert.Equal(t, []string{"a", "c"}, missing)

	// Test all found
	actual = []string{"a", "b", "c"}
	missing = i.getMissingNames(expected, actual)
	assert.Equal(t, []string(nil), missing)

	// Test none found
	actual = []string{"d", "e"}
	missing = i.getMissingNames(expected, actual)
	assert.Equal(t, []string{"a", "b", "c"}, missing)
}

func TestImportTags(t *testing.T) {
	db := mocks.NewDatabase()

	tagNames := []string{existingTagName, missingTagName}

	// Test successful import
	db.Tag.On("FindByNames", testCtx, tagNames, false).Return([]*models.Tag{
		{ID: existingTagID, Name: existingTagName},
	}, nil).Once()
	db.Tag.On("Create", testCtx, mock.AnythingOfType("*models.CreateTagInput")).Run(func(args mock.Arguments) {
		tag := args.Get(1).(*models.CreateTagInput)
		tag.Tag.ID = existingTagID + 1
	}).Return(nil).Once()

	tags, err := importTags(testCtx, db.Tag, tagNames, models.ImportMissingRefEnumCreate)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(tags))

	// Test fail behavior
	db.Tag.On("FindByNames", testCtx, tagNames, false).Return([]*models.Tag{
		{ID: existingTagID, Name: existingTagName},
	}, nil).Once()

	tags, err = importTags(testCtx, db.Tag, tagNames, models.ImportMissingRefEnumFail)
	assert.NotNil(t, err)
	assert.Nil(t, tags)

	// Test ignore behavior
	db.Tag.On("FindByNames", testCtx, tagNames, false).Return([]*models.Tag{
		{ID: existingTagID, Name: existingTagName},
	}, nil).Once()

	tags, err = importTags(testCtx, db.Tag, tagNames, models.ImportMissingRefEnumIgnore)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(tags))

	db.AssertExpectations(t)
}

func TestCreateTags(t *testing.T) {
	db := mocks.NewDatabase()

	tagNames := []string{"tag1", "tag2"}

	db.Tag.On("Create", testCtx, mock.AnythingOfType("*models.CreateTagInput")).Run(func(args mock.Arguments) {
		tag := args.Get(1).(*models.CreateTagInput)
		tag.Tag.ID = 100
	}).Return(nil).Once()
	db.Tag.On("Create", testCtx, mock.AnythingOfType("*models.CreateTagInput")).Run(func(args mock.Arguments) {
		tag := args.Get(1).(*models.CreateTagInput)
		tag.Tag.ID = 101
	}).Return(nil).Once()

	tags, err := createTags(testCtx, db.Tag, tagNames)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(tags))
	assert.Equal(t, "tag1", tags[0].Name)
	assert.Equal(t, "tag2", tags[1].Name)

	db.AssertExpectations(t)
}

func TestCreateTagsError(t *testing.T) {
	db := mocks.NewDatabase()

	tagNames := []string{"tag1"}

	db.Tag.On("Create", testCtx, mock.AnythingOfType("*models.CreateTagInput")).Return(errors.New("Create error")).Once()

	tags, err := createTags(testCtx, db.Tag, tagNames)
	assert.NotNil(t, err)
	assert.Nil(t, tags)

	db.AssertExpectations(t)
}
