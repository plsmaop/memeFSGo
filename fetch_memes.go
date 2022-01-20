package main

import (
	"encoding/json"
	"fmt"
	"log"
	"memefsGo/model"
	"net/http"
	"net/url"
	"strings"
)

var defaultClient = http.Client{}
var knownExt = map[string]bool{"png": true, "jpg": true, "jpeg": true, "mp4": true, "webm": true}

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

		if !knownExt[ext] {
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
		if err != nil {
			log.Println(err)
			continue
		}

		defer resp.Body.Close()

		parsedPosts = append(parsedPosts, model.Post{
			Title: title,
			Url:   rawUrl,
			Size:  uint64(resp.ContentLength),
		})
	}

	return parsedPosts
}

func FetchPosts(c *model.Config) []model.Post {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/.json?limit=%v", c.Subreddit, c.Limit), nil)
	if err != nil {
		log.Println(err)
		return []model.Post{}
	}

	// this is a workaround, otherwise reddit server will return 429
	req.Header.Set("User-Agent", "PostmanRuntime/7.28.4")

	resp, err := defaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return []model.Post{}
	}

	defer resp.Body.Close()

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
