package wpanime

import (
	"fmt"
	"strings"

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
	c := colly.NewCollector()
	episodes := []sources.EpisodeInfo{}

	c.OnRequest(func(r *colly.Request) {
		logger.Logger.Printf("[Episodes] Fetching from URL: %s\n", r.URL.String())
	})

	c.OnHTML("div.eplister > ul > li > a", func(e *colly.HTMLElement) {
		url := e.Attr("href")
		title := e.ChildText(".epl-title")
		number := e.ChildText(".epl-num")
		date := e.ChildText(".epl-date")

		episodes = append(episodes, sources.EpisodeInfo{
			Title:    title,
			URL:      url,
			Number:   number,
			Date:     date,
			Provider: "default",
		})
	})

	c.OnScraped(func(r *colly.Response) {
		logger.Logger.Printf("[Episodes] Found total episodes: %d\n", len(episodes))
	})

	return episodes, c.Visit(showURL)
}

func (w *WPAnimeSource) GetStreamProviders(episodeURL string) ([]sources.StreamProvider, error) {
	c := colly.NewCollector()
	providers := []sources.StreamProvider{}

	c.OnRequest(func(r *colly.Request) {
		logger.Logger.Printf("[Providers] URL: %s\n", r.URL.String())
	})

	c.OnHTML("div.video-players iframe", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if src != "" {
			provider := "unknown"
			if strings.Contains(src, "rumble.com") {
				provider = "rumble"
			}
			providers = append(providers, sources.StreamProvider{
				Name:      provider,
				EmbedURL:  src,
				Processor: provider,
			})
		}
	})

	// Add response logging
	c.OnScraped(func(r *colly.Response) {
		fmt.Printf("Found %d stream providers\n", len(providers))
	})

	return providers, c.Visit(episodeURL)
}
