package webdav

import (
	"fengqi/kodi-metadata-tmdb-cli/config"
	"fengqi/kodi-metadata-tmdb-cli/utils"

	"github.com/studio-b12/gowebdav"
)

// webdav客户端
var client *gowebdav.Client

// webdav电影文件夹根目录
var webDAVMoviesDir string

// webdav剧集文件夹根目录
var webDAVShowsDir string

// 初始化webdav客户端
func InitWebDAV(config *config.WebDAVConfig) {
	url := config.WebDAVUrl
	user := config.WebDAVUser
	pass := config.WebDAVPass
	webDAVMoviesDir = config.MoviesDir
	webDAVShowsDir = config.ShowsDir
	if url == "" || user == "" || pass == "" ||
		webDAVMoviesDir == "" || webDAVShowsDir == "" {
		utils.Logger.Fatal("请先配置webdav!")
		return
	}
	client = gowebdav.NewClient(url, user, pass)
	err := client.Connect()
	if err != nil {
		utils.Logger.Error("webdav连接失败!请检查用户名密码是否正确")
		return
	}
}

// 通过webdav方式删除电影文件夹
func RemoveMovie(movieName string) error {
	err := client.RemoveAll(webDAVMoviesDir + "/" + movieName)
	return err
}

// 通过webdav方式删除剧集文件夹
func RemoveShow(showName string) error {
	err := client.RemoveAll(webDAVShowsDir + "/" + showName)
	return err
}
