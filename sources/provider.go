package sources

type ShowInfo struct {
	Title       string
	URL         string
	Description string
	Thumbnail   string
}

type EpisodeInfo struct {
	Title    string
	URL      string
	Number   string // Changed to string since episode numbers might include prefixes
	Date     string // Added date field
	Provider string
}

type StreamProvider struct {
	Name      string
	EmbedURL  string
	Processor string
}

type SourceProvider interface {
	Search(term string) ([]ShowInfo, error)
	GetEpisodes(showURL string) ([]EpisodeInfo, error)
	GetStreamProviders(episodeURL string) ([]StreamProvider, error)
}
