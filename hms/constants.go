package hms

type MusicSource int

const (
	SOURCE_SPOTIFY MusicSource = iota
	SOURCE_SONGLINK
	SOURCE_YOUTUBE
)

func (s MusicSource) String() string {
	switch s {
	case SOURCE_SPOTIFY:
		return "spotify"
	case SOURCE_SONGLINK:
		return "songlink"
	case SOURCE_YOUTUBE:
		return "youtube"
	default:
		return "unknown"
	}
}
