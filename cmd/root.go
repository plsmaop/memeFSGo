/*
Copyright © 2022 plsmaop allenivan@gmail.com

*/
package cmd

import (
	"log"
	"memefsGo/memefs"
	"memefsGo/model"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	mountpoint  string
	subreddit   string
	limit       int
	refreshSecs int
	debug       bool
)

func init() {
	rootCmd.Flags().StringVarP(&mountpoint, "mountpoint", "m", "", "Mountpoint to mount the folder")
	rootCmd.MarkFlagRequired("mountpoint")

	rootCmd.Flags().StringVarP(&subreddit, "subreddit", "s", "https://www.reddit.com/user/Hydrauxine/m/memes", "Pick a subreddit or multi (requires subreddit URL)")
	rootCmd.Flags().IntVarP(&limit, "limit", "l", 20, "How many memes to fetch at once")
	rootCmd.Flags().IntVarP(&refreshSecs, "refresh_secs", "r", 600, "How often to refresh your memes in secs")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Debug mode")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "memefsGo",
	Short: "A silly user space file system that fetchs memes from subreddits and mount them into your folder",
	Long: `MemeFS is a useless user space file system.
This application fetches memes from the given subreddit periodically and mount them into the given folder.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		fs := memefs.New(model.MemeFSConfig{
			Mountpoint:  mountpoint,
			Subreddit:   subreddit,
			WorkerNum:   20,
			Limit:       limit,
			RefreshSecs: refreshSecs,
			Debug:       debug,
		})

		defer fs.Unmount()

		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sig
			log.Println("MemeFS stop......")
			fs.Unmount()
		}()

		if err := fs.Mount(); err != nil {
			log.Println(err)
			return nil
		}

		return nil
	},
}

func unmountWhenFUSEPanic(fs *memefs.MemeFS) {
	if r := recover(); r != nil {
		log.Println("panic:", r)
		fs.Mount()
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
