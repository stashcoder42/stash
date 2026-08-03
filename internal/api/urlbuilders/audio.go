package urlbuilders

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

type AudioURLBuilder struct {
	BaseURL   string
	AudioID   string
	Checksum  string
	UpdatedAt string
}

func NewAudioURLBuilder(baseURL string, audio *models.Audio) AudioURLBuilder {
	return AudioURLBuilder{
		BaseURL:   baseURL,
		AudioID:   strconv.Itoa(audio.ID),
		Checksum:  audio.Checksum,
		UpdatedAt: strconv.FormatInt(audio.UpdatedAt.Unix(), 10),
	}
}

func (b AudioURLBuilder) GetStreamURL(apiKey string) *url.URL {
	u, err := url.Parse(fmt.Sprintf("%s/audio/%s/stream", b.BaseURL, b.AudioID))
	if err != nil {
		// shouldn't happen
		panic(err)
	}

	if apiKey != "" {
		v := u.Query()
		v.Set("apikey", apiKey)
		u.RawQuery = v.Encode()
	}
	return u
}

func (b AudioURLBuilder) GetCoverURL() string {
	return b.BaseURL + "/audio/" + b.AudioID + "/cover?t=" + b.UpdatedAt
}

func (b AudioURLBuilder) GetThumbnailURL() string {
	return b.BaseURL + "/audio/" + b.AudioID + "/thumbnail?t=" + b.UpdatedAt
}

func (b AudioURLBuilder) GetCaptionPath() string {
	return "/audio/" + b.AudioID + "/caption"
}

func (b AudioURLBuilder) GetCaptionURL() string {
	return b.BaseURL + b.GetCaptionPath()
}
