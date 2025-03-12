package wpanime

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/umesh-verma/anigo/logger"
	"github.com/umesh-verma/anigo/sources"
)

type WPAnimeSource struct {
	BaseURL    string
	SearchPath string
}

func New(baseURL, searchPath string) *WPAnimeSource {
	return &WPAnimeSource{
		BaseURL:    baseURL,
		SearchPath: searchPath,
	}
}

func (w *WPAnimeSource) Search(term string) ([]sources.ShowInfo, error) {
	c := colly.NewCollector()
	shows := []sources.ShowInfo{}

	c.OnRequest(func(r *colly.Request) {
		logger.Logger.Printf("[Search] URL: %s\n", r.URL.String())
	})

	c.OnHTML(".bs > div > a", func(e *colly.HTMLElement) {
		title := e.Attr("title")
		if title == "" {
			title = e.Attr("oldtitle")
		}
		url := e.Attr("href")
		logger.Logger.Printf("[Search] Found: Title=%s, URL=%s\n", title, url)

		shows = append(shows, sources.ShowInfo{
			Title:       title,
			URL:         url,
			Description: strings.TrimSpace(e.Text),
			Thumbnail:   e.ChildAttr("img", "src"),
		})
	})

	c.OnScraped(func(r *colly.Response) {
		logger.Logger.Printf("[Search] Total results: %d\n", len(shows))
	})

	searchURL := w.BaseURL + w.SearchPath + "?s=" + term
	return shows, c.Visit(searchURL)
}

func (w *WPAnimeSource) GetEpisodes(showURL string) ([]sources.EpisodeInfo, error) {
	c := colly.NewCollector(
		colly.Async(true), // Enable async processing
		colly.MaxDepth(1),
	)
	episodes := []sources.EpisodeInfo{}
	var mu sync.Mutex

	c.OnRequest(func(r *colly.Request) {
		logger.Logger.Printf("[Episodes] Fetching from URL: %s\n", r.URL.String())
	})

	c.OnHTML("div.eplister > ul > li > a", func(e *colly.HTMLElement) {
		url := e.Attr("href")
		title := e.ChildText(".epl-title")
		number := e.ChildText(".epl-num")
		date := e.ChildText(".epl-date")

		mu.Lock()
		episodes = append(episodes, sources.EpisodeInfo{
			Title:    title,
			URL:      url,
			Number:   number,
			Date:     date,
			Provider: "default",
		})
		mu.Unlock()
	})

	c.OnScraped(func(r *colly.Response) {
		logger.Logger.Printf("[Episodes] Found total episodes: %d\n", len(episodes))
	})

	err := c.Visit(showURL)
	if err != nil {
		return nil, err
	}

	c.Wait()
	return episodes, nil
}

type mirrorOption struct {
	label string
	value string
}

func cleanEmbedURL(rawURL string) string {
	if strings.HasPrefix(rawURL, "//") {
		return "https:" + rawURL
	}
	if !strings.HasPrefix(rawURL, "http") {
		return "https://" + rawURL
	}
	return rawURL
}

func extractEmbedURL(decodedStr string) string {
	srcIndex := strings.Index(decodedStr, "src=")
	if srcIndex == -1 {
		return ""
	}

	// Find quote character
	quoteChar := byte('"')
	if srcIndex+5 < len(decodedStr) && decodedStr[srcIndex+4] == '\'' {
		quoteChar = '\''
	}

	// Extract URL between quotes
	startIndex := srcIndex + 5
	endIndex := strings.IndexByte(decodedStr[startIndex:], quoteChar)
	if endIndex == -1 {
		return ""
	}

	return cleanEmbedURL(decodedStr[startIndex : startIndex+endIndex])
}

func detectProvider(embedURL, label string) (name, processor string) {
	urlLower := strings.ToLower(embedURL)
	switch {
	case strings.Contains(urlLower, "rumble.com"):
		return "Rumble", "rumble"
	case strings.Contains(urlLower, "youtube.com"), strings.Contains(urlLower, "youtu.be"):
		return "YouTube", "youtube"
	case strings.Contains(urlLower, "dailymotion.com"):
		return "Dailymotion", "dailymotion"
	case strings.Contains(urlLower, "vimeo.com"):
		return "Vimeo", "vimeo"
	default:
		return label, "default"
	}
}

func (w *WPAnimeSource) GetStreamProviders(episodeURL string) ([]sources.StreamProvider, error) {
	c := colly.NewCollector()
	var mirrors []mirrorOption
	providers := []sources.StreamProvider{}

	c.OnRequest(func(r *colly.Request) {
		logger.Logger.Printf("[Providers] URL: %s\n", r.URL.String())
	})

	// Collect mirror options
	c.OnHTML("select.mirror", func(e *colly.HTMLElement) {
		e.ForEach("option", func(_ int, el *colly.HTMLElement) {
			mirrors = append(mirrors, mirrorOption{
				label: el.Text,
				value: el.Attr("value"),
			})
		})
	})

	err := c.Visit(episodeURL)
	if err != nil {
		return nil, err
	}

	// Process mirrors
	for i, mirror := range mirrors {
		// Skip first option if it's a placeholder
		if i == 0 && (mirror.value == "" || mirror.value == "#") {
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(mirror.value)
		if err != nil {
			logger.Logger.Printf("[Providers] Error decoding mirror %s: %v\n", mirror.label, err)
			continue
		}

		embedURL := extractEmbedURL(string(decoded))
		if embedURL == "" {
			continue
		}

		name, processor := detectProvider(embedURL, mirror.label)
		providers = append(providers, sources.StreamProvider{
			Name:      name,
			EmbedURL:  embedURL,
			Processor: processor,
		})

		logger.Logger.Printf("[Providers] Found: %s (%s)\n", name, embedURL)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no valid stream providers found for: %s", episodeURL)
	}

	return providers, nil
}
