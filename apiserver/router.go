package apiserver

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"status-feed-server/entity"
	fc "status-feed-server/feedconst"
	"status-feed-server/prometheus"
	"status-feed-server/ytsource"
)

type ApiController struct {
	*gin.Engine
}

func NewApiController(r *gin.Engine) *ApiController {
	return &ApiController{r}
}

func (router *ApiController) Router() {
	//Prometheus
	router.Use(prometheus.RecordSuccess())
	router.Handle("GET", "/metrics", gin.WrapH(promhttp.Handler()))

	//keeper
	router.Handle("GET", "/", Health)
	router.Handle("GET", "/ping", Health)
	router.Handle("POST", "/api/gets3info", GetS3Info)
	router.Handle("POST", "/api/checksource", CheckSource)
	router.Handle("POST", "/api/recordsource", RecordSource)
	router.Handle("POST", "/api/countview", CountView)
	router.Handle("POST", "/api/downloadview", DownloadView)
	router.Handle("GET", "/api/getrank", GetRank)
	router.Handle("GET", "/api/getdownloadrank", GetDownloadRank)
	router.Handle("GET", "/api/gettoprank", GetTopRank)
	router.Handle("GET", "/api/gettoprankpage", GetTopRankPage)
	router.Handle("POST", "/api/updateshowflag", UpdateShowFlag)
	router.Handle("POST", "/api/gettiktokresource", GetTikTokResource)
	router.Handle("GET", "/api/loadtttodb", LoadTTToDb)
	router.Handle("POST", "/api/exportfiltermd5", ExportFilterMD5)
	router.Handle("GET", "/api/loadtagtttodb", LoadTagTTToDb)
	router.Handle("GET", "/api/loadcountrytagtttoredis", LoadCountryTagTTToRedis) //加载国家tag飞书文档到reids（全量更新非增量）
	router.Handle("GET", "/api/gettttag", GetTTTag)
	router.Handle("GET", "/api/getttbytag", GetTTByTag)
	router.Handle("GET", "/api/getiurlistbytag", GetIURListByTag)
	router.Handle("GET", "/api/getiurtaglist", GetIURTagList)
	router.Handle("GET", "/api/operatevaluetag", OperateValueTag)         //status添加删除展示tag
	router.Handle("GET", "/api/operatetagheadphoto", OperateTagHeadPhoto) //status添加删除tag头图列表
	router.Handle("POST", "/api/recordop", RecordOperation)

	//以下接口是给mod 使用，拉取tiktok video资源
	router.Handle("GET", "/api/loadttvideotodb", LoadTTVideoToDb)
	router.Handle("POST", "/api/gettiktokvideo", GetTikTokVideo)

	//以下接口都是提供给ytmp3 app
	router.Handle("GET", "/api/app/getrank/:loc", AppGetRank)
	router.Handle("POST", "/api/app/getsource", GetYTBSource)
	router.Handle("GET", "/api/app/updatesource", UpdateSource)

	//以下接口都是提供给mp3juices app
	router.Handle("GET", "/api/mp3juices/getresource", GetMusicResource)
	router.Handle("GET", "/api/mp3juices/musiccategory", GetMusicCategory)
	router.Handle("GET", "/api/mp3juices/v2/musiccategory", NewGetMusicCategory)
	router.Handle("GET", "/api/mp3juices/getnewweek", GetNewWeek)
	router.Handle("GET", "/api/mp3juices/loadnewweek", LoadNewWeek)
	router.Handle("GET", "/api/mp3juices/loadbanner", LoadTableContent)
	router.Handle("GET", "/api/mp3juices/getmostartist", GetMostArtist)
	router.Handle("GET", "/api/mp3juices/getmusicbyartist", GetMusicByArtist)
	router.Handle("GET", "/api/mp3juices/loadtopartistresource", LoadTopArtist)

	//以下接口都是提供给wallpaper 旧版本
	router.Handle("GET", "/api/wallpaper/tag", GetTagInfoOld)
	router.Handle("GET", "/api/wallpaper/category", GetCategoryInfoOld)
	router.Handle("GET", "/api/wallpaper/staticbytag", GetStaticInfoByTagOld)
	router.Handle("GET", "/api/wallpaper/staticbycategory", GetStaticInfoByCategoryOld)
	router.Handle("GET", "/api/wallpaper/live", GetLiveInfoOld)
	router.Handle("GET", "/api/wallpaper/livebytag", GetLiveInfoByTagOld)
	router.Handle("GET", "/api/wallpaper/livecategoryid", GetLiveInfoByCategoryIDOld)

	//以下接口都是提供给wallpaper 新版本
	router.Handle("GET", "/api/paper/taginfo", GetTagInfo)
	router.Handle("GET", "/api/paper/categoryinfo", GetCategoryInfo)
	router.Handle("GET", "/api/paper/staginfo", GetStaticInfoByTag)
	router.Handle("GET", "/api/paper/scinfo", GetStaticInfoByCategory)
	router.Handle("GET", "/api/paper/live", GetLiveInfo)
	router.Handle("GET", "/api/paper/ltaginfo", GetLiveInfoByTag)
	router.Handle("GET", "/api/paper/lcinfo", GetLiveInfoByCategoryID)
	router.Handle("GET", "/api/wallpaper/reloadsheet", ReloadSheetInfo)

	router.Handle("GET", "/api/wallpaper/unsplash", UpdateUnsplashPhoto)
	router.Handle("GET", "api/wallpaper/pexels", UpdatePexelsPhoto)

	//以下接口都是对seo需求
	router.Handle("GET", "/api/seoresource/getrankchannel", GetRankChannel)
	router.Handle("GET", "/api/seoresource/getchannelinfo", GetChannelInfo)

	router.Handle("GET", "/api/seoresource/loadresource", LoadResource)

	//提供给google收录mp3juices地址
	router.Handle("GET", "/sitemap.xml", GetSitemapContent)
	router.Handle("GET", "/mp3juciesredirect", Mp3JuicesRedirect)

	//给mp3批量音乐页面
	//router.Handle("GET", "/musicpage/loadlocalsource", LoadMusicPageInfoIntoFeishu)
	//router.Handle("GET", "/musicpage/loadsource", LoadMusicPageInfo)
	router.Handle("GET", "/musicpage/getsource", GetMusicPageInfo)

	//给ytmp3批量页面
	//router.Handle("GET", "/ytmusicpage/loadlocalsource", LoadMusicPageInfoIntoFeishu)
	//router.Handle("GET", "/ytmusicpage/loadsource", LoadYtMusicPageInfo)
	router.Handle("GET", "/ytmusicpage/getsource", GetYtMusicPageInfo)

	//做sticker图片转化
	router.Handle("POST", "/sticker/transform", Transform)

}

func LoadMusicPageInfoIntoFeishu(ctx *gin.Context) {
	//go ytsource.LoadHitSongsPageInfoToFeishu()
	go ytsource.LoadYtHitSongsPageInfoToFeishu()

	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)
}

func HttpSuccessResp(ctx *gin.Context, httpCode int, msg string, data interface{}) {
	resp := entity.HttpResponseObject{
		Status:  fc.HttpSuccessful,
		Message: msg,
		ErrCode: fc.DefaultErrCode,
		Data:    data,
	}

	ctx.JSON(httpCode, resp)
}

func HttpFailResp(ctx *gin.Context, httpCode int, errCode int, data interface{}) {
	resp := entity.HttpResponseObject{
		Status:  fc.HttpFailure,
		Message: fc.ErrorHandler[errCode],
		ErrCode: errCode,
		Data:    data,
	}
	ctx.JSON(httpCode, resp)
}
