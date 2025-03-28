package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/gocolly/colly"
	"github.com/umesh-verma/anigo/streams"
)

const baseURL = "https://www.Donghuastream.org"

type Show struct {
	Title   string
	ShowURL string
}

type Episode struct {
	URL   string
	Title string
}

type MirrorOption struct {
	Label string
	Value string
}
type ProviderInfo struct {
	Name     string
	EmbedURL string
}

type VideoQuality struct {
	URL         string
	Indicator   string
	Score       int
	Width       int
	Height      int
	Resolution  string
	Label       string
	ActualCheck bool
}

func main() {

	//search for a show
	var search string
	fmt.Println("Enter search term:")
	fmt.Scanln(&search)
	search = "/?s=" + search
	showURL, showName := getShows(search)
	episodeURL, episodeTitle := getEpisodes(showURL, showName)

	//get streaming URL
	streamURL, err := getStreamingURL(episodeURL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nStreaming URL for %s: %s\n", episodeTitle, streamURL)

}

func getEpisodes(showURL string, showName string) (string, string) {
	c := colly.NewCollector()
	allEpisodes := []Episode{}

	c.OnRequest(func(r *colly.Request) { fmt.Printf("\nFetching %s episodes...", showName) })
	c.OnError(func(r *colly.Response, err error) { fmt.Println("Something went wrong:", err) })
	c.OnResponse(func(r *colly.Response) { fmt.Println("\nFound all episodes") })

	c.OnHTML("div.eplister > ul > li > a", func(e *colly.HTMLElement) {
		allEpisodes = append(allEpisodes, Episode{
			URL:   e.Attr("href"),
			Title: e.ChildText(".epl-title"),
		})
	})
	c.Visit(showURL)

	slices.Reverse(allEpisodes)
	for i, episode := range allEpisodes {
		fmt.Printf("%d: %s\n", i+1, episode.Title)
	}

	var selectedEpisode int
	fmt.Print("\nSelect episode number: ")
	fmt.Scanln(&selectedEpisode)

	if selectedEpisode > 0 && selectedEpisode <= len(allEpisodes) {
		return allEpisodes[selectedEpisode-1].URL, allEpisodes[selectedEpisode-1].Title
	} else {
		fmt.Println("Invalid episode selection")
		return "", ""
	}

}

func htmlExcerpt(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
func getStreamingURL(episodeURL string) (string, error) {
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	optionsList := []MirrorOption{}

	// Debug callbacks
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting episode page:", r.URL)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Got response from episode page:", r.StatusCode)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error visiting episode page:", err)
	})

	// Get the dropdown value from episode page
	c.OnHTML("select.mirror", func(e *colly.HTMLElement) {

		e.ForEach("option", func(i int, el *colly.HTMLElement) {
			value := el.Attr("value")
			label := el.Text
			optionsList = append(optionsList, MirrorOption{
				Label: label,
				Value: value,
			})

		})
	})

	// Debug handler for HTML content
	c.OnHTML("html", func(e *colly.HTMLElement) {
		if strings.Contains(e.Request.URL.String(), "#debugpage") {
			fmt.Println("Page HTML excerpt:", htmlExcerpt(e.Text, 500))
		}
	})

	err := c.Visit(episodeURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch episode page: %v", err)
	}

	if len(optionsList) == 0 {
		// Try to dump page HTML for debugging
		c.OnHTML("html", func(e *colly.HTMLElement) {
			fmt.Println("Page HTML:", e.Text)
		})
		return "", fmt.Errorf("no mirror options found")
	}

	// Print decoded values for all options and extract provider info
	fmt.Println("\n--- Available Providers ---")

	providers := make([]ProviderInfo, 0, len(optionsList))

	for i, opt := range optionsList {
		if decoded, err := base64.StdEncoding.DecodeString(opt.Value); err == nil {
			decodedStr := string(decoded)

			// Extract src attribute using a simple string search approach
			srcIndex := strings.Index(decodedStr, "src=")
			var embedURL string
			if srcIndex != -1 {
				// Find quote character (either ' or ")
				quoteChar := byte('"')
				if srcIndex+5 < len(decodedStr) && decodedStr[srcIndex+4] == '\'' {
					quoteChar = '\''
				}

				// Extract the URL between quotes
				startIndex := srcIndex + 5 // 'src=' plus quote character
				endIndex := strings.IndexByte(decodedStr[startIndex:], quoteChar)
				if endIndex != -1 {
					embedURL = decodedStr[startIndex : startIndex+endIndex]
				}
			}

			// Ensure URL has proper protocol
			if strings.HasPrefix(embedURL, "//") {
				embedURL = "https:" + embedURL
			}

			// Determine provider based on URL or label
			providerName := opt.Label
			if strings.Contains(embedURL, "rumble.com") {
				providerName = "Rumble"
			} else if strings.Contains(embedURL, "youtube.com") || strings.Contains(embedURL, "youtu.be") {
				providerName = "YouTube"
			} else if strings.Contains(embedURL, "dailymotion.com") {
				providerName = "Dailymotion"
			} else if strings.Contains(embedURL, "vimeo.com") {
				providerName = "Vimeo"
			}

			// Add to providers list
			providers = append(providers, ProviderInfo{
				Name:     providerName,
				EmbedURL: embedURL,
			})
			// If this is the first provider, remove it
			if i == 0 {
				providers = providers[1:]
			} else {
				fmt.Printf("%d: %s\n", i, providerName)
			}

		} else {
			fmt.Printf("%d: %s -> [Error decoding: %v]\n", i, opt.Label, err)
			providers = append(providers, ProviderInfo{
				Name: opt.Label,
			})
		}
	}
	var selectedProvider int
	fmt.Print("\nSelect provider number: ")
	fmt.Scanln(&selectedProvider)

	if selectedProvider < 1 || selectedProvider > len(providers) {
		fmt.Println("Invalid selection")
	}

	// Get the selected provider info
	selected := providers[selectedProvider-1]
	fmt.Printf("\nSelected provider: %s\n", selected.Name)

	// Process based on provider type
	switch {
	case strings.EqualFold(selected.Name, "Rumble"):
		return streams.ProcessRumbleEmbed(selected.EmbedURL)
	default:
		return processGenericEmbed(selected.EmbedURL)

	}
}

func getShows(search string) (showURL string, showName string) {
	url := baseURL + search

	c := colly.NewCollector()
	searchResult := []Show{}
	c.OnRequest(func(r *colly.Request) { fmt.Printf("Looking for %s\n", search[4:]) })
	c.OnError(func(r *colly.Response, err error) { fmt.Println("\nSomething went wrong:", err) })
	c.OnResponse(func(r *colly.Response) { fmt.Printf("\nFetching shows from %s\n\n", baseURL) })
	c.OnHTML(".bs > div > a", func(e *colly.HTMLElement) {
		searchResult = append(searchResult, Show{
			Title:   e.Attr("title"),
			ShowURL: e.Attr("href"),
		})
	})
	c.Visit(url)
	for i, show := range searchResult {
		fmt.Printf("%d %s, \n", i, show.Title)
	}
	var selectedShow int

	fmt.Println("Select a show by number:")
	_, err := fmt.Scanln(&selectedShow)
	if err != nil || selectedShow < 0 || selectedShow >= len(searchResult) {
		log.Fatalf("Invalid selection: %v", err)
	}
	showName1 := searchResult[selectedShow].Title
	showURL1 := searchResult[selectedShow].ShowURL
	return showURL1, showName1
}

func processGenericEmbed(embedURL string) (string, error) {
	return embedURL, nil
}
