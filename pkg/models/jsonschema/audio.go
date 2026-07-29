package jsonschema

import (
	"fmt"
	"os"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/models/json"
)

type AudioFile struct {
	ModTime    json.JSONTime `json:"mod_time,omitempty"`
	Size       string        `json:"size"`
	Duration   string        `json:"duration"`
	AudioCodec string        `json:"audio_codec"`
	Format     string        `json:"format"`
	Bitrate    int           `json:"bitrate"`
	SampleRate int           `json:"sample_rate"`
	Channels   int           `json:"channels"`
}

type Audio struct {
	Title     string `json:"title,omitempty"`
	URL       string `json:"url,omitempty"`
	Date      string `json:"date,omitempty"`
	Details   string `json:"details,omitempty"`
	Rating    int    `json:"rating,omitempty"`
	Organized bool   `json:"organized,omitempty"`

	Performers []string      `json:"performers,omitempty"`
	Tags       []string      `json:"tags,omitempty"`
	Files      []string      `json:"files,omitempty"`
	CreatedAt  json.JSONTime `json:"created_at,omitempty"`
	UpdatedAt  json.JSONTime `json:"updated_at,omitempty"`
}

func (a Audio) Filename(id int, basename string, hash string) string {
	ret := fsutil.SanitiseBasename(a.Title)
	if ret == "" {
		ret = basename
	}

	if hash != "" {
		ret += "." + hash
	} else {
		// audio may have no file and therefore no hash
		ret += "." + strconv.Itoa(id)
	}

	return ret + ".json"
}

func LoadAudioFile(filePath string) (*Audio, error) {
	var audio Audio
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	jsonParser := json.NewDecoder(file)
	err = jsonParser.Decode(&audio)
	if err != nil {
		return nil, err
	}
	return &audio, nil
}

func SaveAudioFile(filePath string, audio *Audio) error {
	if audio == nil {
		return fmt.Errorf("audio must not be nil")
	}
	return marshalToFile(filePath, audio)
}
