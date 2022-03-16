package memefs

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"memefsGo/model"
	"net/http"
	"net/url"
	"strings"
)

var knownExt = map[string]bool{"png": true, "jpg": true, "jpeg": true, "mp4": true, "webm": true}
var defaultClient = http.Client{}

func parsePosts(posts []interface{}) []model.Post {
	parsedPosts := []model.Post{}
	for _, post := range posts {
		data, ok := post.(map[string]interface{})["data"].(map[string]interface{})
		if !ok {
			continue
		}

		rawUrl, ok := data["url"].(string)
		if !ok {
			continue
		}

		parsedUrl, err := url.Parse(rawUrl)
		if err != nil {
			log.Println(err)
			continue
		}

		urlSeg := strings.Split(parsedUrl.Path, "/")
		extSeg := strings.Split(urlSeg[len(urlSeg)-1], ".")

		// no extension
		if len(extSeg) == 1 {
			continue
		}

		ext := extSeg[len(extSeg)-1]

		if _, ok := knownExt[ext]; !ok {
			continue
		}

		title, ok := data["title"].(string)
		if !ok {
			continue
		}

		req, err := http.NewRequest("GET", rawUrl, nil)
		if err != nil {
			log.Println(err)
			continue
		}

		// this is a workaround, otherwise reddit server will return 429
		req.Header.Set("User-Agent", "PostmanRuntime/7.28.4")

		resp, err := defaultClient.Do(req)
		defer resp.Body.Close()
		if err != nil {
			log.Println(err)
			continue
		}

		parsedPosts = append(parsedPosts, model.Post{
			Title: title,
			Url:   rawUrl,
			Size:  uint64(resp.ContentLength),
		})
	}

	return parsedPosts
}

func fetchPosts(c *model.MemeFSConfig) []model.Post {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/.json?limit=%v", c.Subreddit, c.Limit), nil)
	if err != nil {
		log.Println(err)
		return []model.Post{}
	}

	// this is a workaround, otherwise reddit server will return 429
	req.Header.Set("User-Agent", "PostmanRuntime/7.28.4")

	resp, err := defaultClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Println(err)
		return []model.Post{}
	}

	var jsonData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonData)

	if err != nil {
		log.Println(err)
		return []model.Post{}
	}

	posts, ok := jsonData["data"].(map[string]interface{})["children"].([]interface{})
	if !ok {
		log.Println(err)
		return []model.Post{}
	}

	return parsePosts(posts)
}

func fetchMeme(url string) ([]byte, bool) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	// this is a workaround, otherwise reddit server will return 429
	req.Header.Set("User-Agent", "PostmanRuntime/7.28.4")

	resp, err := defaultClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Println(err)
		return nil, false
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	return data, true
}
