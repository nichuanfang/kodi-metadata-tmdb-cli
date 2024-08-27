# kodi-metadata-tmdb-cli

电影、电视剧刮削器命令行版本，使用 TMDB 数据源生成 Kodi 兼容的 NFO 文件和相关图片，可用来代替 Kodi 自带以及 tinyMediaManager 等其他第三方的刮削器。

有定时扫描扫描、实时监听新增文件两种模式，可配置有新增时触发 Kodi 更新媒体库。

# 怎么使用

1. 打开 Kodi 设置 - 媒体 - 视频 - 更改内容（仅限电影和剧集类型） - 信息提供者改为：Local information only
2. 根据平台[下载](https://github.com/nichuanfang/kodi-metadata-tmdb-cli/releases)对应的文件，配置 `config.json`并后台运行。
3. 按照以下结构组织媒体库

    ```shell
    ├─movies                电影目录
    ├─music_videos          音乐剧目录
    ├─shows                 剧集目录
    ├─temp                  临时目录,用于重命名,文件整理
    └─tmm                   temp目录整理好之后放到该目录进行刮削
        ├─movies
        ├─music_videos
        └─shows
    ```

> 本程序必须和下载软件（如 Transmission、µTorrent 等）运行在同一个环境，不然实时监听模式不生效。
> 详细配置参考 [配置总览](https://github.com/fengqi/kodi-metadata-tmdb-cli/wiki/%E9%85%8D%E7%BD%AE%E6%96%87%E4%BB%B6)

# 注意事项

-   升级`clouddrive2`或者正常开机后手动重启`clouddrive2`会导致监听的目录自动移除 需要重新启动刮削器来触发监听
-   需要配置 webdav 来保证剧集合集正常刮削

# 功能列表

-   [x] 从 TMDB 获取电视剧、电视剧分集、电视剧合集、电视剧剧集组、电影、电影合集信息
-   [x] 从 TMDB 获取演员列表、封面图片、海报图片、内容分级、logo
-   [x] 定时扫描电影、电视剧、音乐视频文件和目录
-   [x] 实时监听新添加的电影、电视剧、音乐视频文件和目录
-   [x] 命名不规范或有歧义的电影、电视剧支持手动指定 id
-   [x] 命名不规范的电视剧支持指定 season
-   [x] 多个电视剧剧集组支持指定分组 id
-   [ ] 多个搜索结果尝试根据特征信息确定
-   [x] 更新 NFO 文件后触发 Kodi 更新数据
-   [x] 支持 .part 和 .!qb 文件
-   [x] 音乐视频文件使用 ffmpeg 提取缩略图和视频音频信息

# 参考

> 本程序部分逻辑借鉴了 tinyMediaManager（TMM）的思路，但并非是抄袭，因为编程语言不同，整体思路也不同。

-   Kodi v19 (Matrix) JSON-RPC API/V12 https://kodi.wiki/view/JSON-RPC_API/v12
-   Kodi v19 (Matrix) NFO files https://kodi.wiki/view/NFO_files
-   Kodi Artwork types https://kodi.wiki/view/Artwork_types
-   TMDB Api Overview https://www.themoviedb.org/documentation/api
-   TMDB Api V3 https://developers.themoviedb.org/3/getting-started/introduction
-   File system notifications for Go https://github.com/fsnotify/fsnotify
-   tinyMediaManager https://gitlab.com/tinyMediaManager/tinyMediaManager

# 感谢

![JetBrains Logo (Main) logo](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)
