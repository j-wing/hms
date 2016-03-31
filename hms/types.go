package hms

type MusicSource int

const (
	SOURCE_UNKNOWN MusicSource = iota
	SOURCE_SPOTIFY
	SOURCE_SONGLINK
	SOURCE_YOUTUBE
)

var sourceStringMap = map[MusicSource]string{
	SOURCE_UNKNOWN:  "other",
	SOURCE_SPOTIFY:  "spotify",
	SOURCE_SONGLINK: "songlink",
	SOURCE_YOUTUBE:  "youtube",
}

var sourceIntMap = map[string]MusicSource{
	"other":    SOURCE_UNKNOWN,
	"spotify":  SOURCE_SPOTIFY,
	"songlink": SOURCE_SONGLINK,
	"youtube":  SOURCE_YOUTUBE,
}

func (s MusicSource) String() string {
	return sourceStringMap[s]
}
