package movies

import (
	"fengqi/kodi-metadata-tmdb-cli/config"
	"fengqi/kodi-metadata-tmdb-cli/kodi"
	"fengqi/kodi-metadata-tmdb-cli/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var collector *Collector

func RunCollector(config *config.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	collector = &Collector{
		config:  config,
		channel: make(chan *Movie, 100),
	}

	collector.initWatcher()
	go collector.runWatcher()
	go collector.runMoviesProcess()
	collector.runCronScan()
}

// 电影信息处理：来源包括cron和inotify监听的
func (c *Collector) runMoviesProcess() {
	utils.Logger.Debug("run movies process")

	for {
		select {
		case dir := <-c.channel:
			utils.Logger.DebugF("receive movies task: %v", dir)

			dir.checkCacheDir()
			detail, err := dir.getMovieDetail()
			if err != nil || detail == nil {
				continue
			}

			if !detail.FromCache || !dir.NfoExist(c.config.Collector.MoviesNfoMode) {
				_ = dir.saveToNfo(detail, c.config.Collector.MoviesNfoMode)
				kodi.Rpc.AddRefreshTask(kodi.TaskRefreshMovie, detail.OriginalTitle)
			}
			err = dir.downloadImage(detail)
			moviesStorageDir := c.config.Collector.MoviesStorageDir
			if c.config.Collector.MoveToStorage && err == nil && moviesStorageDir != "" {
				err = dir.MoveToStorage(moviesStorageDir, detail.BelongsToCollection.Name, fmt.Sprintf("%s (%s)",
					utils.SanitizeFileName(detail.Title),
					strings.SplitN(detail.ReleaseDate, "-", 2)[0]))
				if err != nil {
					utils.Logger.ErrorF("移动电影: %s 到存储目录失败: %v", dir.OriginTitle, err)
				}
			}
		}
	}
}

// 运行定时扫描
func (c *Collector) runCronScan() {
	utils.Logger.DebugF("run movies scan cron_seconds: %d", c.config.Collector.CronSeconds)

	task := func() {
		for _, item := range c.config.Collector.MoviesDir {
			// 监听顶级目录
			c.watchDir(item)

			movieDirs, err := c.scanDir(item)
			if err != nil {
				utils.Logger.FatalF("scan movies dir: %s err :%v", item, err)
				continue
			}

			for _, movieDir := range movieDirs {
				c.channel <- movieDir
			}
		}

		if c.config.Kodi.CleanLibrary {
			kodi.Rpc.AddCleanTask("")
		}
	}

	task()
	ticker := time.NewTicker(time.Second * time.Duration(c.config.Collector.CronSeconds))
	for {
		select {
		case <-ticker.C:
			task()
		}
	}
}

// 扫描普通目录，返回其中的电影
func (c *Collector) scanDir(dir string) ([]*Movie, error) {
	movieDirs := make([]*Movie, 0)

	if f, err := os.Stat(dir); err != nil || !f.IsDir() {
		return movieDirs, nil
	}
	dirEntry, err := os.ReadDir(dir)
	if err != nil {
		utils.Logger.ErrorF("scan dir: %s err: %v", dir, err)
		return nil, err
	}

	for _, entry := range dirEntry {
		fileName := entry.Name()
		fileInfo, _ := entry.Info()
		// 合集，以 Iron.Man.2008-2013.Blu-ray.x264.MiniBD1080P-CMCT 为例，暂定使用 2008-2013 做为判断特征
		if yearRange := utils.IsYearRangeLike(fileName); yearRange != "" {
			movieDir, err := c.scanDir(filepath.Join(dir, fileName))
			if err != nil {
				utils.Logger.ErrorF("scan collection dir: %s err: %v", filepath.Join(dir, fileName), err)
				continue
			}
			movieDirs = append(movieDirs, movieDir...)
			continue
		}
		movieDir := parseMoviesDir(dir, fileInfo)
		if movieDir == nil {
			continue
		}
		movieDirs = append(movieDirs, movieDir)
	}

	return movieDirs, nil
}

func (c *Collector) skipFolders(path, filename string) bool {
	base := filepath.Base(path)
	for _, item := range c.config.Collector.SkipFolders {
		if item == base || item == filename {
			return true
		}
	}
	return false
}

func (c *Collector) listFilesAndFolders(path string) []os.FileInfo {
	list := make([]os.FileInfo, 0)
	dirEntry, err := os.ReadDir(path)
	if err != nil {
		return list
	}
	for _, entry := range dirEntry {
		fileName := entry.Name()
		if entry.IsDir() && c.skipFolders(path, fileName) {
			continue
		}
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}
		list = append(list, fileInfo)
	}

	return list
}
