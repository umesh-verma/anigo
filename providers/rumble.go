package providers

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly"
)

func processRumbleEmbed(embedURL string) (string, error) {

	c := colly.NewCollector()

	var jsonData string
	var err error
	videoURL := embedURL
	c.OnHTML("script", func(e *colly.HTMLElement) {
		body := e.Text
		if strings.Contains(body, "\"ua\":{\"mp4\":") {
			startIdx := strings.Index(body, "\"ua\":{\"mp4\":")
			endIdx := strings.Index(body[startIdx:], ",\"timeline\":{")
			if endIdx != -1 {
				jsonData = body[startIdx : startIdx+endIdx]

			}
		}
	})
	print(jsonData)
	//parse the json data in jsonData
	// var videoURL string
	// if jsonData != "" {
	// 	// Complete the JSON structure for parsing
	// 	jsonData = "{" + jsonData + "}"

	// 	// Use encoding/json to parse the data
	// 	type Meta struct {
	// 		Bitrate int `json:"bitrate"`
	// 		Size    int `json:"size"`
	// 		W       int `json:"w"`
	// 		H       int `json:"h"`
	// 	}

	// 	type Resolution struct {
	// 		URL  string `json:"url"`
	// 		Meta Meta   `json:"meta"`
	// 	}

	// 	type VideoData struct {
	// 		UA struct {
	// 			MP4 map[string]Resolution `json:"mp4"`
	// 		} `json:"ua"`
	// 	}

	// 	var data VideoData
	// 	err = json.Unmarshal([]byte(jsonData), &data)
	// 	if err != nil {
	// 		err = fmt.Errorf("failed to parse Rumble JSON: %w", err)
	// 		return "", err
	// 	}

	// 	// Try to get highest quality video (prefer 1080p)
	// 	resolutionPreference := []string{"1080", "720", "480", "360", "240"}
	// 	for _, res := range resolutionPreference {
	// 		if resolution, ok := data.UA.MP4[res]; ok {
	// 			videoURL = resolution.URL
	// 			break
	// 		}
	// 	}

	// 	// If no preferred resolution found, take the first available
	// 	if videoURL == "" && len(data.UA.MP4) > 0 {
	// 		for _, resolution := range data.UA.MP4 {
	// 			videoURL = resolution.URL
	// 			break
	// 		}
	// 	}
	// }

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

	return videoURL, nil

}

func getVideoQuality(url string) string {
	// Simple mapping of quality indicators to resolutions
	if strings.Contains(url, "jaa") {
		return "4K (jaa)"
	} else if strings.Contains(url, "iaa") {
		return "1440p (iaa)"
	} else if strings.Contains(url, "haa") {
		return "1080p (haa)"
	} else if strings.Contains(url, "gaa") {
		return "720p (gaa)"
	} else if strings.Contains(url, "faa") {
		return "480p (faa)"
	} else if strings.Contains(url, "eaa") {
		return "360p (eaa)"
	} else if strings.Contains(url, "daa") {
		return "240p (daa)"
	} else if strings.Contains(url, "caa") {
		return "144p (caa)"
	} else if strings.Contains(url, "baa") {
		return "Low quality (baa)"
	} else if strings.Contains(url, "aaa") {
		return "Lowest quality (aaa)"
	}
	return "Unknown quality"
}
