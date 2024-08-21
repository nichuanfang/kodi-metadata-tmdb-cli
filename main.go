package main

import (
	"fengqi/kodi-metadata-tmdb-cli/config"
	"fengqi/kodi-metadata-tmdb-cli/ffmpeg"
	"fengqi/kodi-metadata-tmdb-cli/kodi"
	"fengqi/kodi-metadata-tmdb-cli/movies"
	"fengqi/kodi-metadata-tmdb-cli/music_videos"
	"fengqi/kodi-metadata-tmdb-cli/shows"
	"fengqi/kodi-metadata-tmdb-cli/tmdb"
	"fengqi/kodi-metadata-tmdb-cli/utils"
	"flag"
	"fmt"
	"runtime"
	"sync"
)

var (
	configFile   string
	version      bool
	buildVersion = "dev-master"
)

func init() {
	flag.StringVar(&configFile, "config", "./config.json", "config file")
	flag.BoolVar(&version, "version", false, "display version")
	flag.Parse()
}

func main() {
	if version {
		fmt.Printf("version: %s, build with: %s\n", buildVersion, runtime.Version())
		return
	}

	c := config.LoadConfig(configFile)

	utils.InitLogger(c.Log.Mode, c.Log.Level, c.Log.File)
	tmdb.InitTmdb(c.Tmdb)
	kodi.InitKodi(c.Kodi)
	ffmpeg.InitFfmpeg(c.Ffmpeg)

	wg := &sync.WaitGroup{}
	wg.Add(3)
	// 刮削电影
	go movies.RunCollector(c, wg)
	//刮削剧集
	go shows.RunCollector(c, wg)
	// 刮削音乐剧
	go music_videos.RunCollector(c, wg)
	wg.Wait()
}
