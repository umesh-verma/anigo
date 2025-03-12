package streams

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gocolly/colly"
)

type VideoData struct {
	URL        string
	Size       int
	Resolution string
}

var videoList []VideoData

func ProcessRumbleEmbed(embedURL string) ([]VideoData, error) {
	c := colly.NewCollector()
	videoList = []VideoData{}
	var err error

	c.OnHTML("script", func(e *colly.HTMLElement) {
		body := e.Text
		var jsonData string
		if strings.Contains(body, "\"ua\":{\"mp4\":") {
			startIdx := strings.Index(body, "\"ua\":{\"mp4\":")
			endIdx := strings.Index(body[startIdx:], ",\"timeline\":{")

			if endIdx != -1 {
				jsonData = body[startIdx+12 : startIdx+endIdx]
			}

			// Temporary map to parse the JSON data
			var tempVideoMap map[string]struct {
				URL  string `json:"url"`
				Meta struct {
					Bitrate int `json:"bitrate"`
					Size    int `json:"size"`
					W       int `json:"w"`
					H       int `json:"h"`
				} `json:"meta"`
			}

			err = json.Unmarshal([]byte(jsonData), &tempVideoMap)
			if err != nil {
				return
			}

			for resolution, data := range tempVideoMap {
				videoList = append(videoList, VideoData{
					URL:        data.URL,
					Size:       data.Meta.Size,
					Resolution: resolution,
				})
			}
		}
	})

	c.OnError(func(r *colly.Response, err2 error) {
		err = fmt.Errorf("failed to scrape Rumble embed: %w", err2)
	})

	err = c.Visit(embedURL)
	if err != nil {
		return nil, err
	}

	if len(videoList) == 0 {
		return nil, fmt.Errorf("no video URLs found in Rumble embed: %s", embedURL)
	}

	c.Wait()
	return videoList, nil
}
