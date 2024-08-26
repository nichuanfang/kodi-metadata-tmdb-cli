package music_videos

import (
	"fengqi/kodi-metadata-tmdb-cli/config"
	"fengqi/kodi-metadata-tmdb-cli/kodi"
	"fengqi/kodi-metadata-tmdb-cli/utils"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

type Collector struct {
	config  *config.Config
	channel chan *MusicVideo
}

var collector *Collector

func RunCollector(config *config.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	collector = &Collector{
		config:  config,
		channel: make(chan *MusicVideo, runtime.NumCPU()),
	}

	collector.initWatcher()
	go collector.runWatcher()
	go collector.runProcessor()
	collector.runScanner()
}

// 处理扫描队列
func (c *Collector) runProcessor() {
	utils.Logger.Debug("run music videos processor")

	limiter := make(chan struct{}, c.config.Ffmpeg.MaxWorker)
	for {
		select {
		case video := <-c.channel:
			utils.Logger.DebugF("receive music video task: %v", video)

			limiter <- struct{}{}
			go func() {
				c.videoProcessor(video)
				<-limiter
			}()
		}
	}
}

// 视频文件处理
func (c *Collector) videoProcessor(video *MusicVideo) {
	if video == nil || (video.NfoExist() && video.ThumbExist()) {
		return
	}

	probe, err := video.getProbe()
	if err != nil {
		utils.Logger.WarningF("parse video %s probe err: %v", filepath.Join(video.Dir, video.OriginTitle), err)
		return
	}

	video.VideoStream = probe.FirstVideoStream()
	video.AudioStream = probe.FirstAudioStream()
	if video.VideoStream == nil || video.AudioStream == nil {
		return
	}

	err = video.drawThumb()
	if err != nil {
		utils.Logger.WarningF("draw thumb err: %v", err)
		return
	}

	err = video.saveToNfo()
	if err != nil {
		utils.Logger.WarningF("save to NFO err: %v", err)
		return
	}

	kodi.Rpc.AddScanTask(video.BaseDir)
}

// 运行扫描器
func (c *Collector) runScanner() {
	utils.Logger.DebugF("run music video scanner cron_seconds: %d", c.config.Collector.CronSeconds)

	task := func() {
		for _, item := range c.config.Collector.MusicVideosDir {
			c.watchDir(item)

			videos, err := c.scanDir(item)
			if len(videos) == 0 || err != nil {
				continue
			}

			// 刮削信息缓存目录
			cacheDir := filepath.Join(item, "tmdb")
			if _, err := os.Stat(cacheDir); err != nil && os.IsNotExist(err) {
				err := os.Mkdir(cacheDir, 0755)
				if err != nil {
					utils.Logger.ErrorF("create probe cache: %s dir err: %v", cacheDir, err)
					continue
				}
			}

			for _, video := range videos {
				c.channel <- video
			}
		}
	}

	task()
	ticker := time.NewTicker(time.Second * time.Duration(c.config.Collector.CronSeconds))
	for range ticker.C {
		task()
		utils.Logger.Debug("run music video scanner finished")
	}
}

func (c *Collector) scanDir(dir string) ([]*MusicVideo, error) {
	videos := make([]*MusicVideo, 0)
	dirInfo, err := func() ([]fs.FileInfo, error) {
		f, err := os.Open(dir)
		if err != nil {
			return nil, err
		}
		list, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			return nil, err
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].Name() < list[j].Name()
		})
		return list, nil
	}()
	if err != nil {
		utils.Logger.WarningF("scanDir %s err: %v", dir, err)
		return nil, err
	}

	for _, file := range dirInfo {
		if file.IsDir() {
			if c.skipFolders(dir, file.Name()) {
				utils.Logger.DebugF("passed in skip folders: %s", file.Name())
				continue
			}

			c.watchDir(filepath.Join(dir, file.Name()))

			subVideos, err := c.scanDir(filepath.Join(dir, file.Name()))
			if err != nil {
				continue
			}

			if len(subVideos) > 0 {
				videos = append(videos, subVideos...)
			}

			continue
		}

		video := c.parseVideoFile(dir, file)
		if video != nil {
			videos = append(videos, video)
		}
	}

	return videos, err
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
