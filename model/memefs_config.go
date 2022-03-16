package model

type MemeFSConfig struct {
	Mountpoint  string
	Subreddit   string
	WorkerNum   int
	Limit       int
	RefreshSecs int
	Debug       bool
}
