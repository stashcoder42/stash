package audio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stashapp/stash/pkg/file/video"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/txn"
)

type CaptionUpdater interface {
	GetAudioCaptions(ctx context.Context, fileID models.FileID) ([]*models.VideoCaption, error)
	UpdateAudioCaptions(ctx context.Context, fileID models.FileID, captions []*models.VideoCaption) error
}

// getCaptionPrefix returns the prefix used to search for audio files for the provided caption path
func getCaptionPrefix(captionPath string) string {
	basename := strings.TrimSuffix(captionPath, filepath.Ext(captionPath)) // caption filename without the extension

	// a caption file can be something like audio_filename.srt or audio_filename.en.srt
	// if a language code is present and valid remove it from the basename
	languageExt := filepath.Ext(basename)
	if len(languageExt) > 2 && video.IsValidLanguage(languageExt[1:]) {
		basename = strings.TrimSuffix(basename, languageExt)
	}

	return basename + "."
}

// getCaptionsLangFromPath returns the language code from a given captions path
// If no valid language is present LangUnknown is returned
func getCaptionsLangFromPath(captionPath string) string {
	langCode := video.LangUnknown
	basename := strings.TrimSuffix(captionPath, filepath.Ext(captionPath)) // caption filename without the extension
	languageExt := filepath.Ext(basename)
	if len(languageExt) > 2 && video.IsValidLanguage(languageExt[1:]) {
		langCode = languageExt[1:]
	}
	return langCode
}

// AssociateCaptions associates captions to audio file(s) with the same basename
func AssociateCaptions(ctx context.Context, captionPath string, txnMgr txn.Manager, fqb models.FileFinder, w CaptionUpdater) {
	captionLang := getCaptionsLangFromPath(captionPath)

	captionPrefix := getCaptionPrefix(captionPath)
	if err := txn.WithTxn(ctx, txnMgr, func(ctx context.Context) error {
		var err error
		files, er := fqb.FindAllByPath(ctx, captionPrefix+"*", true)

		if er != nil {
			return fmt.Errorf("searching for audio %s: %w", captionPrefix, er)
		}

		for _, f := range files {
			// filter out non audio files
			switch f.(type) {
			case *models.AudioFile:
				break
			default:
				continue
			}

			fileID := f.Base().ID
			path := f.Base().Path

			logger.Debugf("Matched captions to audio file %s", path)
			captions, er := w.GetAudioCaptions(ctx, fileID)
			if er == nil {
				fileExt := filepath.Ext(captionPath)
				ext := fileExt[1:]
				if !video.IsLangInCaptions(captionLang, ext, captions) { // only update captions if language code is not present
					newCaption := &models.VideoCaption{
						LanguageCode: captionLang,
						Filename:     filepath.Base(captionPath),
						CaptionType:  ext,
					}
					captions = append(captions, newCaption)
					er = w.UpdateAudioCaptions(ctx, fileID, captions)
					if er == nil {
						logger.Debugf("Updated captions for audio file %s. Added %s", path, captionLang)
					}
				}
			}
		}
		return err
	}); err != nil {
		logger.Error(err.Error())
	}
}

// CleanCaptions removes non existent/accessible language codes from captions
func CleanCaptions(ctx context.Context, f *models.AudioFile, txnMgr txn.Manager, w CaptionUpdater) error {
	captions, err := w.GetAudioCaptions(ctx, f.ID)
	if err != nil {
		return fmt.Errorf("getting captions for audio file %s: %w", f.Path, err)
	}

	if len(captions) == 0 {
		return nil
	}

	filePath := f.Path

	changed := false
	var newCaptions []*models.VideoCaption

	for _, caption := range captions {
		captionPath := caption.Path(filePath)
		_, err := os.Stat(captionPath)
		if errors.Is(err, os.ErrNotExist) {
			logger.Infof("Removing non existent caption %s for %s", caption.Filename, f.Path)
			changed = true
		} else {
			// other errors are ignored for the purposes of cleaning
			newCaptions = append(newCaptions, caption)
		}
	}

	if changed {
		fn := func(ctx context.Context) error {
			return w.UpdateAudioCaptions(ctx, f.ID, newCaptions)
		}

		// possible that we are already in a transaction and txnMgr is nil
		// in that case just call the function directly
		if txnMgr == nil {
			err = fn(ctx)
		} else {
			err = txn.WithTxn(ctx, txnMgr, fn)
		}

		if err != nil {
			return fmt.Errorf("updating captions for audio file %s: %w", f.Path, err)
		}
	}

	return nil
}
