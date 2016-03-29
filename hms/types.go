package hms

import (
	"encoding/json"
)

type MusicSource int

const (
	SOURCE_UNKNOWN MusicSource = iota
	SOURCE_SPOTIFY
	SOURCE_SONGLINK
	SOURCE_YOUTUBE
)

var sourceStringMap = map[MusicSource]string{
	SOURCE_UNKNOWN:  "unknown",
	SOURCE_SPOTIFY:  "spotify",
	SOURCE_SONGLINK: "songlink",
	SOURCE_YOUTUBE:  "youtube",
}

var sourceIntMap = map[string]MusicSource{
	"unknown":  SOURCE_UNKNOWN,
	"spotify":  SOURCE_SPOTIFY,
	"songlink": SOURCE_SONGLINK,
	"youtube":  SOURCE_YOUTUBE,
}

func (s MusicSource) String() string {
	return sourceStringMap[s]
}

func (s MusicSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}
