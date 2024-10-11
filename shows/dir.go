package shows

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fengqi/kodi-metadata-tmdb-cli/tmdb"
	"fengqi/kodi-metadata-tmdb-cli/utils"
	"fengqi/kodi-metadata-tmdb-cli/webdav"
)

var videoExtensions = []string{"mkv",
	"mp4",
	"ts",
	"avi",
	"wmv",
	"m4v",
	"flv",
	"webm",
	"mpeg",
	"mpg",
	"3gp",
	"3gpp",
	"ts",
	"iso",
	"mov",
	"rmvb"}

// Dir 电视剧目录详情，从名字分析
// World.Heritage.In.China.E01-E38.2008.CCTVHD.x264.AC3.720p-CMCT
type Dir struct {
	Dir          string `json:"dir"`
	OriginTitle  string `json:"origin_title"`  // 原始文件名
	Title        string `json:"title"`         // 从视频提取的文件名 鹰眼 Hawkeye
	AliasTitle   string `json:"alias_title"`   // 别名，通常没有用
	ChsTitle     string `json:"chs_title"`     // 分离出来的中文名称 鹰眼
	EngTitle     string `json:"eng_title"`     // 分离出来的英文名称 Hawkeye
	TvId         int    `json:"tv_id"`         // TMDb tv id
	GroupId      string `json:"group_id"`      // TMDB Episode Group
	Season       int    `json:"season"`        // 第几季 ，电影类 -1
	SeasonRange  string `json:"season_range"`  // 合集：S01-S05
	Year         int    `json:"year"`          // 年份：2020、2021
	YearRange    string `json:"year_range"`    // 年份：2010-2015
	Format       string `json:"format"`        // 格式：720p、1080p
	Source       string `json:"source"`        // 来源
	Studio       string `json:"studio"`        // 媒体
	IsCollection bool   `json:"is_collection"` // 是否是合集目录
	PartMode     int    `json:"part_mode"`     // 分卷模式: 0不使用分卷, 1-自动, 2以上为手动指定分卷数量
}

// ReadTvId 从文件读取tvId
func (d *Dir) ReadTvId() {
	idFile := filepath.Join(d.GetCacheDir(), "id.txt")
	if _, err := os.Stat(idFile); err == nil {
		bytes, err := os.ReadFile(idFile)
		if err == nil {
			d.TvId, _ = strconv.Atoi(strings.Trim(string(bytes), "\r\n "))
		} else {
			utils.Logger.WarningF("read tv id specially file: %s err: %v", idFile, err)
		}
	}
}

// CacheTvId 缓存tvId到文件
func (d *Dir) CacheTvId() {
	idFile := filepath.Join(d.GetCacheDir(), "id.txt")
	err := os.WriteFile(idFile, []byte(strconv.Itoa(d.TvId)), 0664)
	if err != nil {
		utils.Logger.ErrorF("save tvId %d to %s err: %v", d.TvId, idFile, err)
	}
}

// ReadSeason 从文件读取季
func (d *Dir) ReadSeason() {
	seasonFile := filepath.Join(d.GetCacheDir(), "season.txt")
	if _, err := os.Stat(seasonFile); err == nil {
		bytes, err := os.ReadFile(seasonFile)
		if err == nil {
			d.Season, _ = strconv.Atoi(strings.Trim(string(bytes), "\r\n "))
		} else {
			utils.Logger.WarningF("read season specially file: %s err: %v", seasonFile, err)
		}
	}

	if d.Season == 0 && len(d.YearRange) == 0 {
		d.Season = 1
	}
}

// ReadGroupId 从文件读取剧集分组
func (d *Dir) ReadGroupId() {
	groupFile := filepath.Join(d.GetCacheDir(), "group.txt")
	if _, err := os.Stat(groupFile); err == nil {
		bytes, err := os.ReadFile(groupFile)
		if err == nil {
			d.GroupId = strings.Trim(string(bytes), "\r\n ")
		} else {
			utils.Logger.WarningF("read group id specially file: %s err: %v", groupFile, err)
		}
	}
}

// GetCacheDir 获取TMDB信息缓存目录, 通常是在每部电视剧的根目录下
func (d *Dir) GetCacheDir() string {
	return filepath.Join(d.GetFullDir(), "tmdb")
}

// GetFullDir 获取电视剧的完整目录
func (d *Dir) GetFullDir() string {
	return filepath.Join(d.Dir, d.OriginTitle)
}

// GetNfoFile 获取电视剧的NFO文件路径
func (d *Dir) GetNfoFile() string {
	return filepath.Join(d.GetFullDir(), "tvshow.nfo")
}

// NfoExist 判断NFO文件是否存在
func (d *Dir) NfoExist() bool {
	nfo := d.GetNfoFile()

	if info, err := os.Stat(nfo); err == nil && info.Size() > 0 {
		return true
	}

	return false
}

// CheckCacheDir 检查并创建缓存目录
func (d *Dir) checkCacheDir() {
	dir := d.GetCacheDir()
	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			utils.Logger.ErrorF("create cache: %s dir err: %v", dir, err)
		}
	}
}

// 下载电视剧的相关图片
// TODO 下载失败后，没有重复以及很长一段时间都不会再触发下载
func (d *Dir) downloadImage(detail *tmdb.TvDetail) {
	utils.Logger.DebugF("download %s images", d.Title)

	if len(detail.PosterPath) > 0 {
		_ = tmdb.DownloadFile(tmdb.Api.GetImageOriginal(detail.PosterPath), filepath.Join(d.GetFullDir(), "poster.jpg"))
	}

	if len(detail.BackdropPath) > 0 {
		_ = tmdb.DownloadFile(tmdb.Api.GetImageOriginal(detail.BackdropPath), filepath.Join(d.GetFullDir(), "fanart.jpg"))
	}

	if detail.Images != nil && len(detail.Images.Logos) > 0 {
		sort.SliceStable(detail.Images.Logos, func(i, j int) bool {
			return detail.Images.Logos[i].VoteAverage > detail.Images.Logos[j].VoteAverage
		})
		image := detail.Images.Logos[0]
		for _, item := range detail.Images.Logos {
			if image.FilePath == "" && item.FilePath != "" {
				image = item
			}
			if item.Iso6391 == "zh" && image.Iso6391 != "zh" {
				image = item
				break
			}
		}
		if image.FilePath != "" {
			logoFile := filepath.Join(d.GetFullDir(), "clearlogo.png")
			_ = tmdb.DownloadFile(tmdb.Api.GetImageOriginal(image.FilePath), logoFile)
		}
	}
}

// 延迟季封面图的下载
// TODO group的信息里可能 season poster不全
func (d *Dir) downloadSeasonPosterImage(detail *tmdb.TvDetail) {
	// TODO group的信息里可能 season poster不全
	if len(detail.Seasons) > 0 {
		for _, item := range detail.Seasons {
			if !d.IsCollection && item.SeasonNumber != d.Season || item.PosterPath == "" {
				continue
			}
			seasonPoster := fmt.Sprintf("season%02d-poster.jpg", item.SeasonNumber)
			_ = tmdb.DownloadFile(tmdb.Api.GetImageOriginal(item.PosterPath), filepath.Join(d.GetFullDir(), seasonPoster))
		}
	}
}

// ReadPart 读取分卷模式
func (d *Dir) ReadPart() {
	partFile := filepath.Join(d.GetCacheDir(), "part.txt")
	if _, err := os.Stat(partFile); err == nil {
		bytes, err := os.ReadFile(partFile)
		if err == nil {
			d.PartMode, _ = strconv.Atoi(strings.Trim(string(bytes), "\r\n "))
		} else {
			utils.Logger.WarningF("read part specially file: %s err: %v", partFile, err)
		}
	}
}

// 刮削完成后 将剧集移动到正式文件夹
func (d *Dir) MoveToStorage(showsStorageDir string, tmdbShowName string, seasonCount int) error {
	// 剧集文件夹
	showDir := filepath.Join(showsStorageDir, tmdbShowName)
	// 季文件夹
	seasonDir := filepath.Join(showDir, fmt.Sprintf("S%02d", seasonCount))
	_, err := os.Stat(showDir)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(showDir, 0755)
	} else if err == nil {
		//遍历剧集目录 找到跟seasonCount相同的季度目录 删除
		dirEntry, err := os.ReadDir(showDir)
		if err != nil {
			return errors.New(err.Error())
		}
		for _, entry := range dirEntry {
			if !entry.IsDir() {
				continue
			}
			showName := entry.Name()
			// 过滤可选字符
			showName = utils.FilterOptionals(showName)
			// 过滤掉或替换歧义的内容
			showName = utils.SeasonCorrecting(showName)
			// 提取季度
			if season := utils.IsSeason(entry.Name()); len(season) > 0 {
				s := season[1:]
				i, err := strconv.Atoi(s)
				if err == nil && i == seasonCount {
					err = os.RemoveAll(filepath.Join(showDir, entry.Name()))
					if err != nil {
						return err
					}
					break
				}
			}
		}

	}
	// 如果剧集tmdb文件夹不存在则 创建tmdb文件夹
	if _, err := os.Stat(filepath.Join(showDir, "tmdb")); err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Join(showDir, "tmdb"), 0755)
		if err != nil {
			return err
		}
	}
	//迁移单季
	_ = moveSingleSeason(filepath.Join(d.Dir, d.OriginTitle), showDir, seasonDir, seasonCount)
	// 季后置处理和日志
	return postProcess(d, tmdbShowName, seasonCount)
}

func moveSingleSeason(fromSeason string, showDir string, toSeason string, seasonCount int) error {
	// Step 1: 整理字幕文件
	_ = processSubtitles(fromSeason)
	// Step 2: 移动剧集元信息文件
	showMetaFiles := []string{
		"tvshow.nfo", "poster.jpg", "fanart.jpg", "clearlogo.png",
		fmt.Sprintf("season%02d-poster.jpg", seasonCount),
		filepath.Join("tmdb", "id.txt"),
		filepath.Join("tmdb", "tv.json"),
	}

	for _, item := range showMetaFiles {
		sourceFile := filepath.Join(fromSeason, item)
		targetFile := filepath.Join(showDir, item)

		// 如果目标文件已存在，删除源文件
		if _, targetErr := os.Stat(targetFile); targetErr == nil {
			if _, sourceErr := os.Stat(sourceFile); sourceErr == nil {
				if err := os.Remove(sourceFile); err != nil {
					fmt.Printf("failed to remove source file %s: %v\n", sourceFile, err)
				}
			}
			continue
		}

		// 移动文件到目标位置
		err := os.Rename(sourceFile, targetFile)
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("failed to move file %s to %s: %v\n", sourceFile, targetFile, err)
		}
	}

	// Step 3: 迁移整个季度文件夹
	_ = os.Rename(fromSeason, toSeason)
	return nil
}

// processSubtitles 处理字幕文件，包括重命名和迁移
func processSubtitles(fromSeason string) error {
	subtitleExtensions := [][]string{
		{".ass"}, // 优先选择 ass 格式
		{".ssa"}, // 然后是 ssa 格式
		{".srt"}, // 最后是 srt 格式
	}
	subtitleDirs := []string{"subs", "subtitles", "字幕"} // 常见字幕文件夹特征

	// Step 1: 获取剧集文件列表
	episodeFiles, err := getEpisodeFiles(fromSeason)
	if err != nil {
		return fmt.Errorf("failed to get episode files: %w", err)
	}
	if len(episodeFiles) == 0 {
		return nil // 如果没有找到剧集文件，直接返回
	}

	// Step 2: 收集所有潜在字幕文件
	allSubtitleFiles := map[string][]string{}

	// (a) 查找字幕文件夹中的字幕文件
	if subtitleDir, _ := findFirstSubtitleFolder(fromSeason, subtitleDirs); subtitleDir != "" {
		for _, extGroup := range subtitleExtensions {
			subtitleFiles, err := getSortedSubtitleFilesByExtension(subtitleDir, extGroup)
			if err == nil && len(subtitleFiles) > 0 {
				allSubtitleFiles[subtitleDir] = subtitleFiles
			}
		}
	}

	// (b) 查找季度文件夹下的字幕文件
	for _, extGroup := range subtitleExtensions {
		subtitleFiles, err := getSortedSubtitleFilesByExtension(fromSeason, extGroup)
		if err == nil && len(subtitleFiles) > 0 {
			allSubtitleFiles[fromSeason] = subtitleFiles
		}
	}

	// 如果没有找到任何字幕文件，提前返回
	if len(allSubtitleFiles) == 0 {
		return nil
	}

	// Step 3: 统一处理所有字幕文件（按优先级匹配剧集文件）
	for dir, subtitleFiles := range allSubtitleFiles {
		minCount := min(len(subtitleFiles), len(episodeFiles))
		for i := 0; i < minCount; i++ {
			newSubtitleName := changeExtension(episodeFiles[i], filepath.Ext(subtitleFiles[i]))
			err := renameAndMoveSubtitle(subtitleFiles[i], dir, newSubtitleName, dir == fromSeason)
			if err != nil {
				fmt.Printf("failed to rename and move subtitle %s: %v\n", subtitleFiles[i], err)
			}
		}
		// 如果是字幕文件夹，则在处理完后删除该文件夹
		if dir != fromSeason {
			_ = os.Remove(dir)
		}
	}

	return nil
}

// findFirstSubtitleFolder 查找第一个符合字幕特征的文件夹
func findFirstSubtitleFolder(fromSeason string, subtitleDirs []string) (string, error) {
	files, err := os.ReadDir(fromSeason)
	if err != nil {
		return "", fmt.Errorf("failed to read season directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			for _, dir := range subtitleDirs {
				if strings.Contains(strings.ToLower(file.Name()), strings.ToLower(dir)) {
					return filepath.Join(fromSeason, file.Name()), nil
				}
			}
		}
	}

	return "", nil
}

// getSortedSubtitleFilesByExtension 获取字幕文件并按文件名排序（只选择指定扩展名的文件）
func getSortedSubtitleFilesByExtension(subtitleDir string, extensions []string) ([]string, error) {
	files, err := os.ReadDir(subtitleDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read subtitle directory: %w", err)
	}

	var subtitleFiles []string
	for _, file := range files {
		if !file.IsDir() && isSubtitleFile(file.Name(), extensions) {
			subtitleFiles = append(subtitleFiles, file.Name())
		}
	}

	// 按文件名排序
	sort.Strings(subtitleFiles)
	return subtitleFiles, nil
}

// getEpisodeFiles 获取季度文件夹中的剧集文件
func getEpisodeFiles(fromSeason string) ([]string, error) {
	files, err := os.ReadDir(fromSeason)
	if err != nil {
		return nil, fmt.Errorf("failed to read season directory: %w", err)
	}

	var episodeFiles []string
	for _, file := range files {
		if !file.IsDir() && isVideoFile(file.Name()) {
			episodeFiles = append(episodeFiles, file.Name())
		}
	}

	// 按文件名排序
	sort.Strings(episodeFiles)
	return episodeFiles, nil
}

// isSubtitleFile 检查文件是否是字幕文件（支持多种扩展名）
func isSubtitleFile(fileName string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}
	return false
}

// isVideoFile 判断文件是否为视频文件 (根据扩展名)
func isVideoFile(fileName string) bool {
	for _, ext := range videoExtensions {
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}
	return false
}

// renameAndMoveSubtitle 重命名并移动字幕文件
func renameAndMoveSubtitle(fileName, fromDir, newFileName string, isFromSeason bool) error {
	oldPath := filepath.Join(fromDir, fileName)
	var newPath string
	if isFromSeason {
		newPath = filepath.Join(fromDir, newFileName)
	} else {
		newPath = filepath.Join(filepath.Dir(fromDir), newFileName)
	}
	err := os.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("failed to rename and move file from %s to %s: %w", oldPath, newPath, err)
	}
	return nil
}

// changeExtension 修改文件名的扩展名
func changeExtension(fileName, newExt string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName)) + newExt
}

// 季处理完毕 如果是合集 当seasonCount等于最大季 则移除合集文件夹
func postProcess(d *Dir, tmdbShowName string, seasonCount int) error {
	if season_range := utils.IsSeasonRange(d.Dir); season_range != "" {
		utils.Logger.InfoF("移动剧集: %s 第%d季 到存储目录成功!", tmdbShowName, seasonCount)
		last_season := strings.Split(season_range, "-")[1]
		s := last_season[1:]
		i, err := strconv.Atoi(s)
		if err == nil && i == seasonCount {
			err = webdav.RemoveShow(filepath.Base(d.Dir))
			if err != nil {
				return err
			}
		}
	}
	utils.Logger.InfoF("移动剧集: %s 存储目录成功!", tmdbShowName)
	return nil
}
