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

func initGetRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// this is a workaround, otherwise reddit server will return 429
	req.Header.Set("User-Agent", "MemeFS Fetcher")
	return req, nil
}

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

		req, err := initGetRequest(rawUrl)
		if err != nil {
			log.Println(err)
			continue
		}

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

func fetchPosts(c *model.MemeFSConfig) []model.Post {
	req, err := initGetRequest(fmt.Sprintf("%s/.json?limit=%v", c.Subreddit, c.Limit))

	resp, err := defaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return []model.Post{}
	}

	if resp.StatusCode != http.StatusOK {
		log.Println(resp.Status)
		return []model.Post{}
	}

	defer resp.Body.Close()

	var jsonData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonData)

	if err != nil {
		log.Println("fetchPosts: decode error:", err)
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
	req, err := initGetRequest(url)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	return data, true
}
