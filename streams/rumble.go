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

func ProcessRumbleEmbed(embedURL string) (string, error) {

	c := colly.NewCollector()

	var err error
	var videoURL string
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

			// Complete the JSON structure for parsing
			err := json.Unmarshal([]byte(jsonData), &tempVideoMap)
			if err != nil {
				fmt.Println("Error parsing JSON:", err)
				return
			}

			// Convert the map to our slice of VideoData
			videoList = []VideoData{}
			for resolution, data := range tempVideoMap {
				videoList = append(videoList, VideoData{
					URL:        data.URL,
					Size:       data.Meta.Size,
					Resolution: resolution,
				})
			}

			fmt.Println("Available Video Qualities:")
			for i, data := range videoList {
				fmt.Printf("%d. Resolution: %sp", i+1, data.Resolution)
				fmt.Printf("  Size: %d MB \n", data.Size/(1024*1024))
			}

			var selectedIndex int
			fmt.Println("Select a video quality (enter number):")
			fmt.Scanln(&selectedIndex)

			if selectedIndex <= 0 || selectedIndex > len(videoList) {
				fmt.Println("Invalid selection")
				return
			}

			videoURL = videoList[selectedIndex-1].URL

		}
	})

	c.OnError(func(r *colly.Response, err2 error) {
		err = fmt.Errorf("failed to scrape Rumble embed: %w", err2)
	})

	err = c.Visit(embedURL)
	if err != nil {
		return "", err
	}

	if videoURL == "" {
		return "", fmt.Errorf("no video URL found in Rumble embed: %s", embedURL)
	}
	c.Wait()
	return videoURL, nil
}
