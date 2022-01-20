/*
Copyright © 2022 plsmaop allenivan@gmail.com
*/
package main

import "memefsGo/model"

func main() {
	// cmd.Execute()
	FetchPosts(&model.Config{
		Subreddit: "https://www.reddit.com/user/Hydrauxine/m/memes",
		Limit:     20,
	})
}
