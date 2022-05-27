package entity

type HttpResponseObject struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	ErrCode int         `json:"err_code"`
	Data    interface{} `json:"data"`
}

type GetS3InfoRequest struct {
	Token string `json:"token" binding:"len=32,required"`
}

type GetS3InfoResponse struct {
	Bucket    string `json:"bucket"`
	SecretID  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
}

type CheckSourceRequest struct {
	FileMD5List []string `json:"file_md5_list" binding:"gt=0,lte=10,required"`
}

type SourceResult struct {
	FileMD5     string `json:"file_md5"`
	IsInfoExist bool   `json:"is_info_exist"`
}

type CheckSourceResponse struct {
	Result []SourceResult `json:"result"`
}

type RecordSourceRequest struct {
	Country  string `json:"country" binding:"gt=0,required"`
	MineType int    `json:"mine_type" binding:"oneof=1 2,required"`
	Source   string `json:"source" binding:"gt=0,required"`
	FileMd5  string `json:"file_md5" binding:"len=32,required"`
	Language string `json:"language"`
}

type CountViewRequest struct {
	FileMD5List []string `json:"file_md5_list" binding:"gt=0,required"`
	Country     string   `json:"country" binding:"gt=0,required"`
	Language    string   `json:"language" binding:"gt=0,required"`
}

type DownloadViewRequest struct {
	FileMD5List []string `json:"file_md5_list" binding:"gt=0,required"`
	Country     string   `json:"country" binding:"gt=0,required"`
	Language    string   `json:"language" binding:"gt=0,required"`
}

type RankObject struct {
	KeeperResourceInfo
	Score float64 `json:"score"`
}

type GetRankResponse struct {
	Result []RankObject `json:"result"`
}

type Ytmp3RankResp struct {
	Result []Ytmp3RankRespItem `json:"result"`
}
type Ytmp3RankRespItem struct {
	Title     string `json:"title"`
	Source    string `json:"source"`
	Score     int64  `json:"score"`
	Duration  string `json:"duration"`
	Thumbnail string `json:"thumbnail"`
}

type YouTuBeVideoRequest struct {
	CategoryID int    `json:"category_id" binding:"oneof=1 2 3 4 5 6,required"`
	Region     string `json:"region" binding:"len=2"`
	Count      int64  `json:"count" binding:"gt=0"`
	Offset     int64  `json:"offset" binding:"gte=0"`
}

type YouTuBeVideoResp struct {
	Result []YouTuBeVideoRespItem `json:"result"`
}

type YouTuBeVideoRespItem struct {
	Title        string `json:"title"`
	Source       string `json:"source"`
	Duration     string `json:"duration"`
	Thumbnail    string `json:"thumbnail"`
	ViewCount    string `json:"viewCount"`
	PublishedAt  string `json:"publishedAt"`
	ChannelTitle string `json:"channelTitle"`
	ChannelId    string `json:"channelId"`
	Avatar       string `json:"avatar"`
}

type GetTagInfoResp struct {
	Result []AdsWallpaperTag `json:"result"`
}

type GetTagInfoRespOld struct {
	Result []AdsWallpaperTagOld `json:"result"`
}

type GetCategoryInfoResp struct {
	Static []AdsWallpaperStaticCategory `json:"static"`
	Live   []AdsWallpaperLiveCategory   `json:"live"`
}

type GetCategoryInfoRespOld struct {
	Static []AdsWallpaperStaticCategoryOld `json:"static"`
	Live   []AdsWallpaperLiveCategoryOld   `json:"live"`
}

type GetStaticInfoRespOld struct {
	Result []AdsWallpaperStaticOld `json:"result"`
}

type GetStaticInfoResp struct {
	Result []AdsWallpaperStatic `json:"result"`
}

type GetLiveInfoResp struct {
	Result []AdsWallpaperLive `json:"result"`
}

type GetLiveInfoRespOld struct {
	Result []AdsWallpaperLiveOld `json:"result"`
}

type LastfmData struct {
	Name   string `json:"name"`
	Url    string `json:"url"`
	Artist string `json:"artist"`
}

type LastfmArtistData struct {
	Avatar string `json:"avatar"`
	Artist string `json:"artist"`
}

type GetArtistDataResp struct {
	Result []LastfmArtistData `json:"result"`
}

type GetLastfmResourceResp struct {
	Result []LastfmData `json:"result"`
}

type CategoryData struct {
	Category    string `json:"category"`
	DisplayName string `json:"display_name"`
	PreviewUrl  string `json:"preview_url"`
	ImageUrl    string `json:"img_url"`
}
type CategoryDataResp struct {
	Genre      []CategoryData        `json:"Genre"`
	RankList   []CategoryData        `json:"Rank List"`
	BannerList []Mp3juicesBannerInfo `json:"Banner"`
}

type NewCategoryDataResp struct {
	Genre      []CategoryData        `json:"Genre"`
	RankList   []CategoryData        `json:"Rank List"`
	BannerList []Mp3juicesBannerInfo `json:"Banner"`
	NewSingle  NewSingleResponse     `json:"New Single"`
}

type GetRankChannelResp struct {
	Result []AdsSEOMP3ChannelInfo `json:"result"`
}

type GetChannelInfoRespResult struct {
	Playlist  AdsSEOMP3PlaylistInfo `json:"playlist"`
	VideoList []AdsSEOMP3VideoInfo  `json:"video_list"`
}

type GetChannelInfoResp struct {
	Result      []GetChannelInfoRespResult `json:"result"`
	ChannelInfo AdsSEOMP3ChannelInfo       `json:"channel_info"`
}

type UpdateShowFlagReq struct {
	FileMd5 string `json:"file_md5" binding:"len=32,required"`
	IsShow  int    `json:"is_show" binding:"lte=1"`
}

type ModActivityObject struct {
	LinkURL string `json:"link_url"`
}

type ModActivityResp struct {
	Result ModActivityObject `json:"result"`
}

type GetTikTokResourceRequest struct {
	UserID  string `json:"user_id" binding:"gt=0,required"`
	Country string `json:"country" binding:"gt=0,required"`
}

type GetTikTokVideoRequest struct {
	UserID  string `json:"user_id" binding:"gt=0,required"`
	Country string `json:"country" binding:"gt=0,required"`
}

type GetTikTokResourceResponse struct {
	Result []TikTokResourceInfo `json:"result"`
}

type GetNewWeekResponse struct {
	Result []AdsNewWeekVideoInfo `json:"result"`
}

type GetImgUrResourceResponse struct {
	List []KeeperStatusResourceData `json:"list"`
	Nums int64                      `json:"nums"`
}

type ExportFilterMD5Request struct {
	FilterMD5List []string `json:"filter_md5_list" binding:"gt=0,lte=10,required"`
}

type GetTikTokVideoResponse struct {
	Result []TikTokVideoInfo `json:"result"`
}

type NewSingleResponse struct {
	DisplayName string               `json:"display_name"`
	MusicList   []NewSingleMusicList `json:"music_list"`
}

type NewSingleMusicList struct {
	Artist      string `json:"artist"`
	Name        string `json:"name"`
	PlaylistURL string `json:"playlist_url"`
	YTBID       string `json:"ytb_id"`
}

type GetTikTokVideoTagResponse struct {
	Result []string `json:"tag_list"`
}

type KeeperStatusResourceData struct {
	ResourceID      string   `json:"resource_id"`
	Author          string   `json:"author"`
	AuthorHeadPhoto string   `json:"author_head_photo"`
	Title           string   `json:"title"`
	URL             string   `json:"url"`
	ResourceType    int8     `json:"type"`
	TagList         []string `json:"tag_list"`
}

type KeeperStatusTag struct {
	Name      string `json:"name"`
	HeadPhoto string `json:"head_photo"`
}

type RecordOperationRequest struct {
	UserID            string `json:"user_id" binding:"gt=0,required"`
	Source            int    `json:"source" binding:"gt=0,required"`
	VideoID           string `json:"video_id" binding:"gt=0,required"`
	ShowCount         int    `json:"show_count"`
	TapCount          int    `json:"tap_count"`
	WatchCount        int    `json:"watch_count"`
	IsCompleteShow    int    `json:"is_complete_show"`
	VideoWaitTime     int    `json:"video_wait_time"`
	VideoWaitTimeStr  string `json:"video_wait_time_str"`
	SendWhatsappCount int    `json:"send_whatsapp_count"`
	ShareCount        int    `json:"share_count"`
	DownloadCount     int    `json:"download_count"`
	ReturnCount       int    `json:"return_count"`
}

type MusicPageInfo struct {
	ID        int    `json:"id"`
	SongTitle string `json:"song_title"`
	YTBTitle  string `json:"ytb_title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	Lyrics    string `json:"lyrics"`
	YTBUrl    string `json:"ytb_url"`
	YTBImg    string `json:"ytb_img"`
}
