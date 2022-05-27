package apiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/go-redis/redis"
	"io/ioutil"
	"net/http"
	"os"
	"status-feed-server/datamanager"
	"status-feed-server/entity"
	fc "status-feed-server/feedconst"
	"status-feed-server/globalctx"
	"status-feed-server/keeper"
	logger "status-feed-server/logger"
	"status-feed-server/prometheus"
	"status-feed-server/sticker"
	"status-feed-server/utils"
	"status-feed-server/wallpaper"
	"status-feed-server/ytsource"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/snabb/sitemap"
)

func Health(ctx *gin.Context) {
	HttpSuccessResp(ctx, http.StatusOK, "status-feed-server health OK", "")
}

func GetS3Info(ctx *gin.Context) {
	var r entity.GetS3InfoRequest

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Warn("ShouldBindWith err", err)
		prometheus.ReportKeeperResourceError("GetS3Info", "GetS3Info", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	cfg := globalctx.GetCfg()
	token := cfg.Section("AWSS3").Key("token").String()
	bucket := cfg.Section("AWSS3").Key("bucket").String()
	secretID := cfg.Section("AWSS3").Key("secret_id").String()
	secretKey := cfg.Section("AWSS3").Key("secret_key").String()

	if r.Token != token {
		prometheus.ReportKeeperResourceError("GetS3Info", "GetS3Info", "token")
		HttpFailResp(ctx, http.StatusOK, fc.ForbidGetS3InfoErrCode, nil)
		return
	}

	logger.Logger.Info("GetS3Info token:", r.Token)

	resp := &entity.GetS3InfoResponse{
		Bucket:    bucket,
		SecretID:  secretID,
		SecretKey: secretKey,
	}

	HttpSuccessResp(ctx, http.StatusOK, "status-feed-server health OK", resp)
}

func CheckSource(ctx *gin.Context) {
	var r entity.CheckSourceRequest

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Warn("ShouldBindWith err", err)
		prometheus.ReportKeeperResourceError("CheckSource", "CheckSource", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	logger.Logger.Info("request params:", r.FileMD5List)

	//TODO：检测数据是否存在，加个缓存，缓存失效去mysql获取。当前的缓存以周为单位
	redisHelper := datamanager.GetRedisInstance()
	sourceList, err := redisHelper.HMGet(fc.KeeperSourceRedisKey, r.FileMD5List...)
	if err != nil {
		logger.Logger.Warn("redis HMGet err:", err)
		prometheus.ReportKeeperResourceError("CheckSource", "CheckSource", "redis HMGet err")
		HttpFailResp(ctx, http.StatusOK, fc.RedisServerErrorCode, nil)
		return
	}

	result := entity.CheckSourceResponse{
		Result: make([]entity.SourceResult, 0, len(r.FileMD5List)),
	}

	for index, md5 := range r.FileMD5List {
		var sr entity.SourceResult
		sr.FileMD5 = md5
		if sourceList[index] == nil {
			sr.IsInfoExist = false
		} else {
			sr.IsInfoExist = true
		}

		result.Result = append(result.Result, sr)
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", result)
}

func RecordSource(ctx *gin.Context) {
	var r entity.RecordSourceRequest

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Warn("ShouldBindWith err:", err)
		prometheus.ReportKeeperResourceError("RecordSource", "RecordSource", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	if keeper.IsFMInFilterList(r.FileMd5) {
		HttpSuccessResp(ctx, http.StatusOK, "exec success", "filter source")
		return
	}

	logger.Logger.WithField("country", r.Country).
		WithField("mine_type", r.MineType).
		WithField("source", r.Source).
		WithField("file_md5", r.FileMd5).
		WithField("language", r.Language).
		Info("request params:")

	r.Source = fc.CDNHttpURL + r.Source

	var buffer bytes.Buffer
	buffer.WriteString(r.Country)
	buffer.WriteString(fc.GapStr)
	buffer.WriteString(strconv.Itoa(r.MineType))
	buffer.WriteString(fc.GapStr)
	buffer.WriteString(r.Source)
	buffer.WriteString(fc.GapStr)
	buffer.WriteString(r.FileMd5)
	buffer.WriteString(fc.GapStr)
	buffer.WriteString(r.Language)

	content := buffer.String()

	//TODO:每周的资源排行榜，增加过期时间
	redisHelper := datamanager.GetRedisInstance()
	_, err = redisHelper.HSet(fc.KeeperSourceRedisKey, r.FileMd5, content)
	if err != nil {
		logger.Logger.Warn("redis HSet err:", err)
		prometheus.ReportKeeperResourceError("RecordSource", "RecordSource", "redis HSet err")
		HttpFailResp(ctx, http.StatusOK, fc.RedisServerErrorCode, nil)
		return
	}
	//redisHelper.Expire(fc.KeeperSourceRedisKey, keeper.RedisExpire)

	var data []entity.KeeperResourceInfo
	tempData := entity.KeeperResourceInfo{
		FileMd5:    r.FileMd5,
		Source:     r.Source,
		Country:    r.Country,
		MineType:   r.MineType,
		Language:   r.Language,
		CreateTime: time.Now().Unix(),
		IsShow:     1,
	}

	data = append(data, tempData)

	err = datamanager.GetMysqlInstance().InsertKeeperResourceInfo(data)
	if err != nil {
		logger.Logger.Warn("InsertKeeperResourceInfo err:", err)

		if strings.Contains(err.Error(), "Duplicate entry") {
			HttpSuccessResp(ctx, http.StatusOK, "exec success", "already exist")
		} else {
			prometheus.ReportKeeperResourceError("RecordSource", "RecordSource", "entry already exist")
			HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		}
		return
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", "")
}

func CountView(ctx *gin.Context) {
	var r entity.CountViewRequest

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		prometheus.ReportKeeperResourceError("CountView", "CountView", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	logger.Logger.Info("request params:", r.FileMD5List)

	keeper.SaveInfoToRedis(r)

	HttpSuccessResp(ctx, http.StatusOK, "exec success", "")
}

func DownloadView(ctx *gin.Context) {
	var r entity.DownloadViewRequest

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		prometheus.ReportKeeperResourceError("DownloadView", "DownloadView", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	logger.Logger.Info("request params:", r.FileMD5List)

	keeper.SaveToRedisDownloadInfo(r)

	HttpSuccessResp(ctx, http.StatusOK, "exec success", "")
}

func GetRank(ctx *gin.Context) {
	c := ctx.Query("country")
	la := ctx.Query("language")

	var key string
	if c != "" {
		key = c
	} else if la != "" {
		key = la
	}

	result := keeper.GetRankInfoByLc(key, false)

	HttpSuccessResp(ctx, http.StatusOK, "exec success", result)
}

func GetDownloadRank(ctx *gin.Context) {
	c := ctx.Query("country")
	//la := ctx.Query("language")
	//
	//var key string
	//if c != "" {
	//	key = c
	//} else if la != "" {
	//	key = la
	//}

	result := keeper.GetRankInfo(c)

	HttpSuccessResp(ctx, http.StatusOK, "exec success", result)
}

func GetTopRank(ctx *gin.Context) {
	c := ctx.Query("country")
	if c == "" {
		prometheus.ReportKeeperResourceError("GetTopRank", "GetTopRank", "country err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
	}
	key := c
	keeper.GetTopRankInfo(key, false)
	HttpSuccessResp(ctx, http.StatusOK, "exec success", "")
}

func GetTopRankPage(ctx *gin.Context) {
	c := ctx.Query("country")
	//枚举country？
	if c == "" {
		prometheus.ReportKeeperResourceError("GetTopRankPage", "GetTopRankPage", "country err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
	}
	pageNum, err := strconv.ParseInt(ctx.Query("pagenum"), 10, 64)
	if err != nil {
		logger.Logger.Warn("parseInt", err)
		prometheus.ReportKeeperResourceError("GetTopRankPage", "GetTopRankPage", "page err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}
	pageSize, err := strconv.ParseInt(ctx.Query("pagesize"), 10, 64)
	if err != nil {
		logger.Logger.Warn("parseInt", err)
		prometheus.ReportKeeperResourceError("GetTopRankPage", "GetTopRankPage", "page err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}
	if pageSize <= 0 || pageNum < 0 {
		prometheus.ReportKeeperResourceError("GetTopRankPage", "GetTopRankPage", "page err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
	}

	//isDownload, err := strconv.ParseBool(ctx.Query("isdownload"))
	//if err != nil {
	//	logger.Logger.Warn("parseBool", err)
	//	HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
	//	return
	//}

	data := keeper.GetTopRankPage(c, pageNum, pageSize, true)
	HttpSuccessResp(ctx, http.StatusOK, "exec success", data)
}

func UpdateShowFlag(ctx *gin.Context) {
	var r entity.UpdateShowFlagReq
	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Warn("ShouldBindWith err", err)
		prometheus.ReportKeeperResourceError("UpdateShowFlag", "UpdateShowFlag", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	logger.Logger.Info("request params:", r.FileMd5)

	//提前检查redis中是否存在数据，防止压力到mysql
	//2022-04-20 修改策略，因为是人工筛选。压力不会很大。所以去掉校验
	//redisHelper := datamanager.GetRedisInstance()
	//resource, _ := redisHelper.HGet(fc.KeeperSourceRedisKey, r.FileMd5)
	//if resource == "" {
	//	logger.Logger.Warn("resource not exist:")
	//	HttpFailResp(ctx, http.StatusOK, fc.ResourceNotExistCode, nil)
	//	return
	//}

	err = datamanager.GetMysqlInstance().UpdateKeeperResourceInfoByMd5(r.FileMd5, r.IsShow)
	if err != nil {
		logger.Logger.Warn("UpdateKeeperResourceInfoByMd5 err:", err)
		prometheus.ReportKeeperResourceError("UpdateShowFlag", "UpdateShowFlag", "UpdateKeeperResourceInfoByMd5 err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)
}

func AppGetRank(ctx *gin.Context) {
	loc := ctx.Param("loc")
	if loc == "" {
		logger.Logger.Error(fc.RequestParamStructErrMsg)
		prometheus.ReportYtmp3ResourceError("AppGetRank", "AppGetRank", "loc err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamStructErrCode, "")
		return
	}

	locMap := map[string]bool{
		"IN": true,
		"US": true,
		"ID": true,
		"PH": true,
		"MY": true,
	}

	if _, ok := locMap[loc]; !ok {
		loc = "US"
	}

	hClient := http.Client{
		Timeout: 5 * time.Second,
	}

	url := fmt.Sprintf("https://154.82.111.45.sslip.io/app/ytbrank/%s", loc)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Logger.Error(fc.NewYtmp3HttpReqErrMsg + ":" + err.Error())
		prometheus.ReportYtmp3ResourceError("AppGetRank", "AppGetRank", "GET request err")
		HttpFailResp(ctx, http.StatusOK, fc.NewYtmp3HttpReqErrCode, "")
		return
	}

	resp, err := hClient.Do(req)
	if err != nil {
		logger.Logger.Error(fc.GetYtmp3RankDataErrMsg + ":" + err.Error())
		prometheus.ReportYtmp3ResourceError("AppGetRank", "AppGetRank", "Client err")
		HttpFailResp(ctx, http.StatusOK, fc.GetYtmp3RankDataErrCode, "")
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.Error(fc.GetYtmp3RankDataErrMsg + ":" + err.Error())
		prometheus.ReportYtmp3ResourceError("AppGetRank", "AppGetRank", "get rank err")
		HttpFailResp(ctx, http.StatusOK, fc.GetYtmp3RankDataErrCode, "")
		return
	}

	result := &entity.Ytmp3RankResp{}
	result.Result = make([]entity.Ytmp3RankRespItem, 0, 20)
	_, err = jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, e error) {
		//fmt.Printf("each, Value: '%s'\t Type: %s\n", string(value), dataType)
		if jsonparser.Object == dataType {
			item := entity.Ytmp3RankRespItem{}
			item.Title, _ = jsonparser.GetString(value, "title")
			item.Source, _ = jsonparser.GetString(value, "key")
			item.Score, _ = jsonparser.GetInt(value, "score")
			item.Duration, _ = jsonparser.GetString(value, "duration")
			item.Thumbnail = utils.GetYouTuBeStandardThumbnail(item.Source)

			item.Source = fc.YouTuBeDefaultURLPrefix + item.Source
			result.Result = append(result.Result, item)
		}
	})
	if err != nil {
		logger.Logger.Error(fc.GetYtmp3RankDataErrMsg + ":" + err.Error())
		prometheus.ReportYtmp3ResourceError("AppGetRank", "AppGetRank", "get rank err")
		HttpFailResp(ctx, http.StatusOK, fc.GetYtmp3RankDataErrCode, "")
		return
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", result)
}

func GetYTBSource(ctx *gin.Context) {
	var r entity.YouTuBeVideoRequest
	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Error(fc.RequestParamErrMsg + ":" + err.Error())
		prometheus.ReportYtmp3ResourceError("GetYTBSource", "GetYTBSource", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	//加载youtube信息
	go ytsource.Init()

	result := ytsource.GetYTBResult(r)
	HttpSuccessResp(ctx, http.StatusOK, "exec success", result)
}

func GetTagInfoOld(ctx *gin.Context) {
	result, err := datamanager.GetMysqlInstance().GetAllTags()
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + ":" + err.Error())
		prometheus.ReportWallpaperOldResourceError("GetTagInfoOld", "GetTagInfoOld", "GetAllTags err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	var r entity.GetTagInfoRespOld
	r.Result = make([]entity.AdsWallpaperTagOld, 0, len(result))
	for _, item := range result {
		var t entity.AdsWallpaperTagOld
		t.TagID = item.TagID
		t.TagName = item.TagName
		t.Display = item.Display
		t.Attr = item.Attr
		t.CreateTime = item.CreateTime

		r.Result = append(r.Result, t)
	}
	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetTagInfo(ctx *gin.Context) {
	result, err := datamanager.GetMysqlInstance().GetAllTags()
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + ":" + err.Error())
		prometheus.ReportWallpaperNewResourceError("GetTagInfo", "GetTagInfo", "GetAllTags err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	var r entity.GetTagInfoResp
	r.Result = result
	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetCategoryInfoOld(ctx *gin.Context) {
	static, err := datamanager.GetMysqlInstance().GetStaticCategory()
	if err != nil {
		prometheus.ReportWallpaperOldResourceError("GetCategoryInfoOld", "GetCategoryInfoOld", "GetStaticCategory err")
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetStaticCategory:" + err.Error())
	}

	live, err := datamanager.GetMysqlInstance().GetLiveCategory()
	if err != nil {
		prometheus.ReportWallpaperOldResourceError("GetCategoryInfoOld", "GetCategoryInfoOld", "GetLiveCategory err")
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetLiveCategory:" + err.Error())
	}

	var r entity.GetCategoryInfoRespOld
	r.Static = make([]entity.AdsWallpaperStaticCategoryOld, 0, len(static))
	r.Live = make([]entity.AdsWallpaperLiveCategoryOld, 0, len(live))
	for _, item := range static {
		var t entity.AdsWallpaperStaticCategoryOld
		t.ImgUrl = item.ImgUrl
		t.Category = item.Category
		t.CategoryID = item.CategoryID
		t.Display = item.Display
		t.CreateTime = item.CreateTime

		r.Static = append(r.Static, t)
	}

	for _, item := range live {
		var t entity.AdsWallpaperLiveCategoryOld
		t.ImgUrl = item.ImgUrl
		t.Category = item.Category
		t.CategoryID = item.CategoryID
		t.Display = item.Display
		t.CreateTime = item.CreateTime

		r.Live = append(r.Live, t)
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetCategoryInfo(ctx *gin.Context) {
	static, err := datamanager.GetMysqlInstance().GetStaticCategory()
	if err != nil {
		prometheus.ReportWallpaperNewResourceError("GetCategoryInfo", "GetCategoryInfo", "GetStaticCategory err")
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetStaticCategory:" + err.Error())
	}

	live, err := datamanager.GetMysqlInstance().GetLiveCategory()
	if err != nil {
		prometheus.ReportWallpaperNewResourceError("GetCategoryInfo", "GetCategoryInfo", "GetLiveCategory err")
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetLiveCategory:" + err.Error())
	}

	var r entity.GetCategoryInfoResp
	r.Static = static
	r.Live = live

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}
func GetStaticInfoByTagOld(ctx *gin.Context) {
	tagIDStr := ctx.Query("tag")
	pageStr := ctx.Query("page")

	tagID, err1 := strconv.Atoi(tagIDStr)
	page, err2 := strconv.Atoi(pageStr)
	if err1 != nil || err2 != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err1, err2)
		prometheus.ReportWallpaperOldResourceError("GetStaticInfoByTagOld", "GetStaticInfoByTagOld", "str err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	dbHelper := datamanager.GetMysqlInstance()
	tagIDArray, err := dbHelper.GetStaticByStaticID(page, wallpaper.PageSize, tagID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetStaticIDByTagID:" + err.Error())
		prometheus.ReportWallpaperOldResourceError("GetStaticInfoByTagOld", "GetStaticInfoByTagOld", "GetStaticByStaticID err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	var newTagIDArray []entity.AdsWallpaperStatic
	for _, item := range tagIDArray {
		if item.ImgUrl == "" {
			continue
		}

		newTagIDArray = append(newTagIDArray, item)
	}

	length := len(newTagIDArray)
	logger.Logger.Info("newTagIDArray len--", len(newTagIDArray))

	if page <= 3 && length < wallpaper.PageSize {
		gap := wallpaper.PageSize - length
		newSourceArray := wallpaper.GetAndSaveStaticResource(gap, page, tagID)
		logger.Logger.Info("newSourceArray--", newSourceArray)

		nsa := len(newSourceArray)
		if nsa < gap {
			gap = nsa
		}

		if len(newSourceArray) > 0 {
			newTagIDArray = append(newTagIDArray, newSourceArray[:gap]...)
		}
	}

	logger.Logger.Info("tagIDArray after len--", len(newTagIDArray))
	var r entity.GetStaticInfoRespOld
	filterMap := make(map[int64]bool)
	for _, item := range newTagIDArray {
		if item.ImgUrl == "" {
			continue
		}

		if _, ok := filterMap[item.StaticID]; ok {
			continue
		}

		var t entity.AdsWallpaperStaticOld
		t.ImgUrl = item.ImgUrl
		t.CategoryID = item.CategoryID
		t.Category = item.Category
		t.StaticID = item.StaticID
		t.Attr = item.Attr
		t.Thumbnail = item.Thumbnail
		t.CreateTime = item.CreateTime

		r.Result = append(r.Result, t)
		filterMap[item.StaticID] = true
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}
func GetStaticInfoByTag(ctx *gin.Context) {
	tagIDStr := ctx.Query("t")
	pageStr := ctx.Query("p")

	tagID, err1 := strconv.Atoi(tagIDStr)
	page, err2 := strconv.Atoi(pageStr)
	if err1 != nil || err2 != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err1, err2)
		prometheus.ReportWallpaperNewResourceError("GetStaticInfoByTag", "GetStaticInfoByTag", "str err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	dbHelper := datamanager.GetMysqlInstance()
	tagIDArray, err := dbHelper.GetStaticByStaticID(page, wallpaper.PageSize, tagID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetStaticIDByTagID:" + err.Error())
		prometheus.ReportWallpaperNewResourceError("GetStaticInfoByTag", "GetStaticInfoByTag", "GetStaticByStaticID err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	var newTagIDArray []entity.AdsWallpaperStatic
	for _, item := range tagIDArray {
		if item.ImgUrl == "" {
			continue
		}

		newTagIDArray = append(newTagIDArray, item)
	}

	length := len(newTagIDArray)
	logger.Logger.Info("newTagIDArray len--", len(newTagIDArray))

	if page <= 3 && length < wallpaper.PageSize {
		gap := wallpaper.PageSize - length
		newSourceArray := wallpaper.GetAndSaveStaticResource(gap, page, tagID)
		logger.Logger.Info("newSourceArray--", newSourceArray)

		nsa := len(newSourceArray)
		if nsa < gap {
			gap = nsa
		}

		if len(newSourceArray) > 0 {
			newTagIDArray = append(newTagIDArray, newSourceArray[:gap]...)
		}
	}

	logger.Logger.Info("tagIDArray after len--", len(newTagIDArray))
	var r entity.GetStaticInfoResp
	filterMap := make(map[int64]bool)
	for _, item := range newTagIDArray {
		if item.ImgUrl == "" {
			continue
		}

		if _, ok := filterMap[item.StaticID]; ok {
			continue
		}

		r.Result = append(r.Result, item)
		filterMap[item.StaticID] = true
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetStaticInfoByCategory(ctx *gin.Context) {
	categoryIDStr := ctx.Query("c_id")
	pageStr := ctx.Query("p")

	categoryID, err1 := strconv.Atoi(categoryIDStr)
	page, err2 := strconv.Atoi(pageStr)
	if err1 != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err1, err2)
		prometheus.ReportWallpaperNewResourceError("GetStaticInfoByCategory", "GetStaticInfoByCategory", "str err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, page)
		return
	}

	if err2 != nil || page <= 0 {
		page = 1
	}

	var r entity.GetStaticInfoResp
	result, err := datamanager.GetMysqlInstance().GetStaticByCategoryID(page, 100, categoryID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetStaticByCategoryID:" + err.Error())
		prometheus.ReportWallpaperNewResourceError("GetStaticInfoByCategory", "GetStaticInfoByCategory", "GetStaticByCategoryID err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, page)
		return
	}

	r.Result = result

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetStaticInfoByCategoryOld(ctx *gin.Context) {
	categoryIDStr := ctx.Query("category_id")
	pageStr := ctx.Query("page")

	categoryID, err1 := strconv.Atoi(categoryIDStr)
	page, err2 := strconv.Atoi(pageStr)
	if err1 != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err1, err2)
		prometheus.ReportWallpaperOldResourceError("GetStaticInfoByCategoryOld", "GetStaticInfoByCategoryOld", "str err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, page)

		return
	}

	if err2 != nil || page <= 0 {
		page = 1
	}

	result, err := datamanager.GetMysqlInstance().GetStaticByCategoryID(page, 100, categoryID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetStaticByCategoryID:" + err.Error())
		prometheus.ReportWallpaperOldResourceError("GetStaticInfoByCategoryOld", "GetStaticInfoByCategoryOld", "GetStaticByCategoryID err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, page)
		return
	}

	var r entity.GetStaticInfoRespOld
	for _, item := range result {
		var t entity.AdsWallpaperStaticOld
		t.ImgUrl = item.ImgUrl
		t.CategoryID = item.CategoryID
		t.Category = item.Category
		t.StaticID = item.StaticID
		t.Attr = item.Attr
		t.Thumbnail = item.Thumbnail
		t.CreateTime = item.CreateTime

		r.Result = append(r.Result, t)
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetLiveInfoOld(ctx *gin.Context) {
	pageStr := ctx.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	result, err := datamanager.GetMysqlInstance().GetLiveInfo(page, 100)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetLiveInfo:" + err.Error())
		prometheus.ReportWallpaperOldResourceError("GetLiveInfoOld", "GetLiveInfoOld", "GetLiveInfo err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, page)
		return
	}

	var r entity.GetLiveInfoRespOld
	r.Result = make([]entity.AdsWallpaperLiveOld, 0, len(result))
	for _, item := range result {
		var t entity.AdsWallpaperLiveOld
		t.VideoUrl = item.VideoUrl
		t.ImgUrl = item.ImgUrl
		t.CategoryID = item.CategoryID
		t.Category = item.Category
		t.LiveID = item.LiveID
		t.Attr = item.Attr
		t.Thumbnail = item.Thumbnail
		t.CreateTime = item.CreateTime

		r.Result = append(r.Result, t)
	}
	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetLiveInfo(ctx *gin.Context) {
	pageStr := ctx.Query("p")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	var r entity.GetLiveInfoResp
	result, err := datamanager.GetMysqlInstance().GetLiveInfo(page, 100)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetLiveInfo:" + err.Error())
		prometheus.ReportWallpaperNewResourceError("GetLiveInfo", "GetLiveInfo", "GetLiveInfo err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, page)
		return
	}

	r.Result = result

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetLiveInfoByCategoryIDOld(ctx *gin.Context) {
	categoryIDStr := ctx.Query("category_id")
	pageStr := ctx.Query("page")

	categoryID, err1 := strconv.Atoi(categoryIDStr)
	page, err2 := strconv.Atoi(pageStr)
	if err1 != nil || err2 != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err1, err2)
		prometheus.ReportWallpaperOldResourceError("GetLiveInfoByCategoryIDOld", "GetLiveInfoByCategoryIDOld", "str err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, page)

		return
	}

	result, err := datamanager.GetMysqlInstance().GetLiveInfoByCategoryID(page, 100, categoryID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetLiveInfo:" + err.Error())
		prometheus.ReportWallpaperOldResourceError("GetLiveInfoByCategoryIDOld", "GetLiveInfoByCategoryIDOld", "GetLiveInfoByCategoryID err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, page)
		return
	}

	var r entity.GetLiveInfoRespOld
	r.Result = make([]entity.AdsWallpaperLiveOld, 0, len(result))
	for _, item := range result {
		var t entity.AdsWallpaperLiveOld
		t.VideoUrl = item.VideoUrl
		t.ImgUrl = item.ImgUrl
		t.CategoryID = item.CategoryID
		t.Category = item.Category
		t.LiveID = item.LiveID
		t.Attr = item.Attr
		t.Thumbnail = item.Thumbnail
		t.CreateTime = item.CreateTime

		r.Result = append(r.Result, t)
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetLiveInfoByCategoryID(ctx *gin.Context) {
	categoryIDStr := ctx.Query("c_id")
	pageStr := ctx.Query("p")

	categoryID, err1 := strconv.Atoi(categoryIDStr)
	page, err2 := strconv.Atoi(pageStr)
	if err1 != nil || err2 != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err1, err2)
		prometheus.ReportWallpaperNewResourceError("GetLiveInfoByCategoryID", "GetLiveInfoByCategoryID", "str err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, page)

		return
	}

	var r entity.GetLiveInfoResp
	result, err := datamanager.GetMysqlInstance().GetLiveInfoByCategoryID(page, 100, categoryID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetLiveInfo:" + err.Error())
		prometheus.ReportWallpaperNewResourceError("GetLiveInfoByCategoryID", "GetLiveInfoByCategoryID", "GetLiveInfoByCategoryID err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, page)
		return
	}

	r.Result = result

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func ReloadSheetInfo(ctx *gin.Context) {
	categoryIDStr := ctx.Query("sheet_name")
	if v, ok := wallpaper.SheetIDMap[categoryIDStr]; ok {
		wallpaper.GetSingleSheetInfo(categoryIDStr, v)
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)
}

func UpdateUnsplashPhoto(ctx *gin.Context) {
	count := ctx.Query("count")
	n, err := strconv.Atoi(count)
	if err != nil {
		logger.Logger.Error("count is not a number")
		return
	}
	for _, category := range wallpaper.PhotoCategory {
		wallpaper.UpdateUnsplashPhoto(category, n)
	}
	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)
}

func UpdatePexelsPhoto(ctx *gin.Context) {
	count := ctx.Query("count")
	n, err := strconv.Atoi(count)
	if err != nil {
		logger.Logger.Error("count is not a number")
		prometheus.ReportWallpaperNewResourceError("UpdatePexelsPhoto", "UpdatePexelsPhoto", "count err")
		return
	}
	for _, category := range wallpaper.PhotoCategory {
		wallpaper.UpdatePexelsPhoto(category, n)
	}
	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)
}

func GetMusicResource(ctx *gin.Context) {
	categoryIDStr := ctx.Query("category")
	if categoryIDStr == "" {
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	result, err := ytsource.GetMusicFromRedis(categoryIDStr)
	if len(result.Result) == 0 {
		prometheus.ReportMp3JuicesResourceError("GetMusicResource", "GetMusicResource", categoryIDStr)
		if err != nil {
			logger.Logger.Error(fc.MysqlExecErrorMsg + " GetMusicFromRedis:" + err.Error())
		}
		HttpFailResp(ctx, http.StatusOK, fc.RedisServerErrorCode, nil)
		return
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", result)
}

func GetMusicCategory(ctx *gin.Context) {
	language := ctx.Query("language")
	result := ytsource.GetYtmp3Category(language)

	bannerInfo, err := datamanager.GetMysqlInstance().GetBannerInfo(5)
	if err != nil {
		prometheus.ReportMp3JuicesResourceError("GetMusicCategory", "GetMusicCategory", "GetBannerInfo err")
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetBannerInfo:" + err.Error())
	}
	result.BannerList = bannerInfo

	go ytsource.GetLastFMResource()

	HttpSuccessResp(ctx, http.StatusOK, "exec success", result)
}

func NewGetMusicCategory(ctx *gin.Context) {
	language := ctx.Query("language")
	result := ytsource.NewGetYtmp3Category(language)

	bannerInfo, err := datamanager.GetMysqlInstance().GetBannerInfo(5)
	if err != nil {
		prometheus.ReportMp3JuicesResourceError("NewGetMusicCategory", "NewGetMusicCategory", "GetBannerInfo err")
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetBannerInfo:" + err.Error())
	}
	result.BannerList = bannerInfo

	newSingle, _ := ytsource.GetNewSingelMusic(language)
	result.NewSingle = newSingle

	go ytsource.GetLastFMResource()

	HttpSuccessResp(ctx, http.StatusOK, "exec success", result)
}

func GetLiveInfoByTagOld(ctx *gin.Context) {
	tagIDStr := ctx.Query("tag")
	pageStr := ctx.Query("page")

	tagID, err1 := strconv.Atoi(tagIDStr)
	page, err2 := strconv.Atoi(pageStr)
	if err1 != nil || err2 != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err1, err2)
		prometheus.ReportWallpaperOldResourceError("GetLiveInfoByTagOld", "GetLiveInfoByTagOld", "str err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	dbHelper := datamanager.GetMysqlInstance()
	tagIDArray, err := dbHelper.GetLiveByStaticID(page, wallpaper.PageSize, tagID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetLiveByStaticID:" + err.Error())
		prometheus.ReportWallpaperOldResourceError("GetLiveInfoByTagOld", "GetLiveInfoByTagOld", "GetLiveByStaticID err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	var newTagIDArray []entity.AdsWallpaperLive
	for _, item := range tagIDArray {
		if item.ImgUrl == "" {
			continue
		}

		newTagIDArray = append(newTagIDArray, item)
	}

	length := len(newTagIDArray)
	logger.Logger.Info("newTagIDArray len--", len(newTagIDArray))

	if page <= 3 && length < wallpaper.PageSize {
		gap := wallpaper.PageSize - length
		newSourceArray := wallpaper.GetAndSaveLiveResource(gap, page, tagID)
		logger.Logger.Info("newSourceArray--", newSourceArray)

		if len(newSourceArray) > 0 {
			newTagIDArray = append(newTagIDArray, newSourceArray...)
		}
	}

	logger.Logger.Info("tagIDArray len--", len(newTagIDArray))
	var r entity.GetLiveInfoRespOld
	filterMap := make(map[int64]bool)
	for _, item := range newTagIDArray {
		if item.VideoUrl == "" {
			continue
		}

		if _, ok := filterMap[item.LiveID]; ok {
			continue
		}

		//fmt.Println(item.LiveID)
		var t entity.AdsWallpaperLiveOld
		t.VideoUrl = item.VideoUrl
		t.ImgUrl = item.ImgUrl
		t.CategoryID = item.CategoryID
		t.Category = item.Category
		t.LiveID = item.LiveID
		t.Attr = item.Attr
		t.Thumbnail = item.Thumbnail
		t.CreateTime = item.CreateTime

		r.Result = append(r.Result, t)
		filterMap[item.LiveID] = true
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetLiveInfoByTag(ctx *gin.Context) {
	tagIDStr := ctx.Query("t")
	pageStr := ctx.Query("p")

	tagID, err1 := strconv.Atoi(tagIDStr)
	page, err2 := strconv.Atoi(pageStr)
	if err1 != nil || err2 != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err1, err2)
		prometheus.ReportWallpaperNewResourceError("GetLiveInfoByTag", "GetLiveInfoByTag", "str err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	dbHelper := datamanager.GetMysqlInstance()
	tagIDArray, err := dbHelper.GetLiveByStaticID(page, wallpaper.PageSize, tagID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetLiveByStaticID:" + err.Error())
		prometheus.ReportWallpaperNewResourceError("GetLiveInfoByTag", "GetLiveInfoByTag", "GetLiveByStaticID err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	var newTagIDArray []entity.AdsWallpaperLive
	for _, item := range tagIDArray {
		if item.ImgUrl == "" {
			continue
		}

		newTagIDArray = append(newTagIDArray, item)
	}

	length := len(newTagIDArray)
	logger.Logger.Info("newTagIDArray len--", len(newTagIDArray))

	if page <= 3 && length < wallpaper.PageSize {
		gap := wallpaper.PageSize - length
		newSourceArray := wallpaper.GetAndSaveLiveResource(gap, page, tagID)
		logger.Logger.Info("newSourceArray--", newSourceArray)

		if len(newSourceArray) > 0 {
			newTagIDArray = append(newTagIDArray, newSourceArray...)
		}
	}

	logger.Logger.Info("tagIDArray len--", len(newTagIDArray))
	var r entity.GetLiveInfoResp
	filterMap := make(map[int64]bool)
	for _, item := range newTagIDArray {
		if item.VideoUrl == "" {
			continue
		}

		if _, ok := filterMap[item.LiveID]; ok {
			continue
		}

		r.Result = append(r.Result, item)
		filterMap[item.LiveID] = true
	}

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetRankChannel(ctx *gin.Context) {
	pageStr := ctx.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		logger.Logger.Error(fc.RequestParamErrMsg, err)
		page = 1
	}

	if page <= 0 {
		page = 1
	}

	var r entity.GetRankChannelResp
	dbHelper := datamanager.GetMysqlInstance()
	result, err := dbHelper.GetAdsSEOMP3ChannelInfo(page, 10)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetAdsSEOMP3ChannelInfo:" + err.Error())
		prometheus.ReportSeoResourceError("GetRankChannel", "GetRankChannel", "GetAdsSEOMP3ChannelInfo err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	r.Result = result
	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func GetChannelInfo(ctx *gin.Context) {
	channelID := ctx.Query("channel_id")
	if channelID == "" {
		prometheus.ReportSeoResourceError("GetChannelInfo", "GetChannelInfo", "channelID err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	var r entity.GetChannelInfoResp
	var result []entity.GetChannelInfoRespResult
	r.Result = result

	dbHelper := datamanager.GetMysqlInstance()
	plResult, err := dbHelper.GetAdsSEOMP3PlaylistInfo(channelID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetAdsSEOMP3PlaylistInfo:" + err.Error())
		prometheus.ReportSeoResourceError("GetChannelInfo", "GetChannelInfo", "GetAdsSEOMP3PlaylistInfo err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	for _, item := range plResult {
		if item.PlaylistID == "" {
			continue
		}

		var tPl entity.GetChannelInfoRespResult
		tPl.Playlist = item

		var videoIDS []string

		plItem, err := dbHelper.GetAdsSEOMP3PlaylistItemInfoByPlayID(channelID, item.PlaylistID)
		if err != nil {
			logger.Logger.Error(fc.MysqlExecErrorMsg + " GetAdsSEOMP3PlaylistItemInfoByPlayID:" + err.Error())
			prometheus.ReportSeoResourceError("GetChannelInfo", "GetChannelInfo", "GetAdsSEOMP3PlaylistItemInfoByPlayID err")
			return
		}

		filterMap := make(map[string]bool)
		for _, pl := range plItem {
			_, ok := filterMap[pl.VideoID]
			if pl.VideoID != "" && !ok {
				videoIDS = append(videoIDS, pl.VideoID)
				filterMap[pl.VideoID] = true
			}
		}

		videoIDInfos, err := dbHelper.GetAdsSEOMP3VideoInfo(videoIDS)
		if err != nil {
			logger.Logger.Error(fc.MysqlExecErrorMsg + " GetAdsSEOMP3VideoInfo:" + err.Error())
			prometheus.ReportSeoResourceError("GetChannelInfo", "GetChannelInfo", "GetAdsSEOMP3VideoInfo err")
			return
		}
		tPl.VideoList = videoIDInfos

		r.Result = append(r.Result, tPl)
	}

	cInfo, err := dbHelper.GetAdsSEOMP3ChannelInfoByCid(channelID)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetAdsSEOMP3ChannelInfoByCid:" + err.Error())
		prometheus.ReportSeoResourceError("GetChannelInfo", "GetChannelInfo", "GetAdsSEOMP3ChannelInfoByCid err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	r.ChannelInfo = cInfo

	HttpSuccessResp(ctx, http.StatusOK, "exec success", r)
}

func LoadResource(ctx *gin.Context) {
	channelID := ctx.Query("passwd")
	if channelID != "ytmp3seo" {
		prometheus.ReportSeoResourceError("LoadResource", "LoadResource", "channelID err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	go ytsource.InitSEOResource()

	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)
}

var globalSiteMapURL = []string{
	"https://www.mp3juices.cc/juice01/",
	"https://www.mp3juices.cc/faq/",
	"https://www.mp3juices.cc/cutter/",
	"https://www.mp3juices.cc/news/",
	"https://www.mp3juices.cc/contact/",
	"https://www.mp3juices.cc/copyright-claims/",
	"https://www.mp3juices.cc/privacy-policy/",
	"https://www.mp3juices.cc/special-rightholders-accounts/",
	"https://www.mp3juices.cc/terms-of-use/",
	"https://www.mp3juices.cc/youtube-to-mp301/",
	"https://www.mp3juices.cc/youtube-converter01/",
	"https://www.mp3juices.cc/my-free-mp301/",
	"https://www.mp3juices.cc/ytmp301/",
	"https://www.mp3juices.cc/savefrom01/",
	"https://www.mp3juices.cc/y2mate01/",
	"https://www.mp3juices.cc/tubidy01/",
}

func Mp3JuicesRedirect(c *gin.Context) {
	dmid, _ := GetJuicesDmcaidFromYtmpCore()
	fmt.Println(dmid.Id, "  ", dmid.LastMod)

	url := "https://www.mp3juices.cc/juices" + dmid.Id
	c.Redirect(http.StatusMovedPermanently, url)

}

func GetSitemapContent(c *gin.Context) {
	sm := sitemap.New()

	dmid, _ := GetJuicesDmcaidFromYtmpCore()

	ts, _ := strconv.ParseInt(dmid.LastMod, 10, 64)
	if ts <= 0 {
		ts = time.Now().Unix()
	}

	t := time.Unix(ts, 0).UTC()
	var p float32

	for index, item := range globalSiteMapURL {
		p = 0.2
		if index == 0 {
			p = 0.9
		}

		loc := strings.Replace(item, "01", dmid.Id, -1)
		sm.Add(&sitemap.URL{
			Loc:        loc,
			LastMod:    &t,
			ChangeFreq: sitemap.Daily,
			Priority:   p,
		})
	}

	sm.WriteTo(c.Writer)
	c.Header("content-type", "text/xml; charset=utf-8")
}

type Ytmp3CoreDmcaIDObject struct {
	Id      string `json:"id"`
	LastMod string `json:"last_mod"`
}

func GetJuicesDmcaidFromYtmpCore() (r Ytmp3CoreDmcaIDObject, err error) {
	url := "https://154.82.111.45.sslip.io/mp3juices/getjuicesid"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Logger.Error(fc.NewYtmp3HttpReqErrMsg + ":" + err.Error())
		return
	}

	hClient := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := hClient.Do(req)
	if err != nil {
		logger.Logger.Error(fc.GetYtmp3RankDataErrMsg + ":" + err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.Error(fc.GetYtmp3RankDataErrMsg + ":" + err.Error())
		return
	}

	r.Id, _ = jsonparser.GetString(body, "data", "id")
	r.LastMod, _ = jsonparser.GetString(body, "data", "last_mod")

	return
}

var ExistTTCountryMap = map[string]bool{
	"MX": true,
	"BR": true,
	"US": true,
	"ID": true,
	"PE": true,
	"PK": true,
	"BO": true,
	"PY": true,
	"NG": true,
	"ZA": true,
	"EC": true,
	"AR": true,
	"GT": true,
	"CO": true,
	"CN": true,
	"TW": true,
}

func GetTikTokResource(ctx *gin.Context) {
	var r entity.GetTikTokResourceRequest

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Warn("ShouldBindWith err", err)
		prometheus.ReportKeeperResourceError("GetTikTokResource", "GetTikTokResource", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	if _, ok := ExistTTCountryMap[r.Country]; !ok {
		HttpSuccessResp(ctx, http.StatusOK, "success", nil)
		return
	}

	timeStr := time.Now().Format("2006-01-02")
	rKey := "tiktok:use:page" + timeStr
	//if redis err or not exist in hash key, index=1
	redisHelper := datamanager.GetRedisInstance()
	prevIndex, _ := redisHelper.HGet(rKey, r.UserID)
	index, _ := strconv.Atoi(prevIndex)
	page := index%10 + 1

	var resp entity.GetTikTokResourceResponse

	result, err := datamanager.GetMysqlInstance().GetTikTokByCountry(r.Country, page, 10)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetTikTokByCountry:" + err.Error())
		prometheus.ReportKeeperResourceError("GetTikTokResource", "GetTikTokResource", "GetTikTokByCountry err")
		HttpSuccessResp(ctx, http.StatusOK, "success", nil)
		return
	}

	resp.Result = result

	if len(resp.Result) == 0 {
		page = 1
		redisHelper.HSet(rKey, r.UserID, page)

		result, err := datamanager.GetMysqlInstance().GetTikTokByCountry(r.Country, page, 10)
		if err != nil {
			logger.Logger.Error(fc.MysqlExecErrorMsg + " GetTikTokByCountry again:" + err.Error())
			prometheus.ReportKeeperResourceError("GetTikTokResource", "GetTikTokResource", "GetTikTokByCountry again")
			HttpSuccessResp(ctx, http.StatusOK, "success", nil)
			return
		}

		resp.Result = result
	}

	redisHelper.HSet(rKey, r.UserID, page)
	redisHelper.Expire(rKey, 24*time.Hour)
	HttpSuccessResp(ctx, http.StatusOK, "success", resp)
}

func LoadTTToDb(ctx *gin.Context) {
	keeper.GetSingleSheetInfo(fc.TikTokResourceSheetID)

	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

func LoadTagTTToDb(ctx *gin.Context) {
	keeper.GetSingleSheetInfo(fc.TikTokTagVideoResourceSheetID)

	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

func LoadTTVideoToDb(ctx *gin.Context) {
	keeper.GetSingleSheetInfo(fc.TikTokVideoResourceSheetID)

	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

//
// LoadCountryTagTTToRedis
// @Author:hhz
// @Date:2022-05-17 19:23:10
// @Description: 加载各国家tag到Redis
// @param ctx *gin.Context
//
func LoadCountryTagTTToRedis(ctx *gin.Context) {
	keeper.GetSingleSheetInfo(fc.TikTokCountryTagSheetID)
	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

func GetNewWeek(ctx *gin.Context) {
	data, err := datamanager.GetMysqlInstance().GetNewWeekVideoInfo()
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetNewWeekVideoInfo err:" + err.Error())
		prometheus.ReportMp3JuicesResourceError("GetNewWeek", "GetNewWeek", "GetNewWeekVideoInfo err")
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
		return
	}

	var result entity.GetNewWeekResponse
	result.Result = data

	HttpSuccessResp(ctx, http.StatusOK, "success", result)
}

func LoadNewWeek(ctx *gin.Context) {
	ytsource.GetNewThisWeekResource()

	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

func LoadTableContent(ctx *gin.Context) {
	ytsource.GetBannerResource()

	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

func ExportFilterMD5(ctx *gin.Context) {
	var r entity.ExportFilterMD5Request

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Warn("ShouldBindWith err", err)
		prometheus.ReportKeeperResourceError("ExportFilterMD5", "ExportFilterMD5", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	filterMap := make(map[string]interface{}, len(r.FilterMD5List))
	for _, v := range r.FilterMD5List {
		if v != "" {
			filterMap[v] = "filter"
		}
	}

	err = keeper.SetFilterList(filterMap)
	if err != nil {
		prometheus.ReportKeeperResourceError("ExportFilterMD5", "ExportFilterMD5", "SetFilterList err")
		HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, "export error"+err.Error())
		return
	}

	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

func GetTikTokVideo(ctx *gin.Context) {
	var r entity.GetTikTokVideoRequest

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Warn("ShouldBindWith err", err)
		prometheus.ReportModResourceError("GetTikTokVideo", "GetTikTokVideo", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	if _, ok := ExistTTCountryMap[r.Country]; !ok {
		HttpSuccessResp(ctx, http.StatusOK, "success", nil)
		return
	}

	timeStr := time.Now().Format("2006-01-02")
	rKey := "tiktok:video:page" + timeStr
	//if redis err or not exist in hash key, index=1
	redisHelper := datamanager.GetRedisInstance()
	prevIndex, _ := redisHelper.HGet(rKey, r.UserID)
	index, _ := strconv.Atoi(prevIndex)
	page := index%10 + 1
	pageSize := 4

	var resp entity.GetTikTokVideoResponse

	result, err := datamanager.GetMysqlInstance().GetTikTokVideoByCountry(r.Country, page, pageSize)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetTikTokVideoByCountry:" + err.Error())
		prometheus.ReportModResourceError("GetTikTokVideo", "GetTikTokVideo", "GetTikTokVideoByCountry err")
		HttpSuccessResp(ctx, http.StatusOK, "success", nil)
		return
	}

	resp.Result = result

	if len(resp.Result) != pageSize {
		page = 1
		redisHelper.HSet(rKey, r.UserID, page)

		result, err := datamanager.GetMysqlInstance().GetTikTokVideoByCountry(r.Country, page, pageSize)
		if err != nil {
			logger.Logger.Error(fc.MysqlExecErrorMsg + " GetTikTokVideoByCountry again:" + err.Error())
			prometheus.ReportModResourceError("GetTikTokVideo", "GetTikTokVideo", "GetTikTokVideoByCountry err")
			HttpSuccessResp(ctx, http.StatusOK, "success", nil)
			return
		}

		resp.Result = result
	}

	redisHelper.HSet(rKey, r.UserID, page)
	redisHelper.Expire(rKey, 24*time.Hour)

	HttpSuccessResp(ctx, http.StatusOK, "success", resp)
}

func GetMostArtist(ctx *gin.Context) {
	result, err := ytsource.GetMostArtistFromRedis()
	if err != nil {
		prometheus.ReportMp3JuicesResourceError("GetMostArtist", "GetMostArtist", "GetMostArtistFromRedis err")
		HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, nil)
		return
	}

	HttpSuccessResp(ctx, http.StatusOK, "success", result)
}

func GetMusicByArtist(ctx *gin.Context) {
	artist := ctx.Query("artist")
	result, err := ytsource.GetMusicByArtist(artist)
	if err != nil {
		prometheus.ReportMp3JuicesResourceError("GetMusicByArtist", "GetMusicByArtist", "GetMusicByArtist err")
		HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, nil)
		return
	}

	HttpSuccessResp(ctx, http.StatusOK, "success", result)
}

func LoadTopArtist(ctx *gin.Context) {
	//ytsource.GetTopArtistInfo()
	ytsource.GetNewSingleVideoInfo()

	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

//
// GetTTTag
// @Author:hhz
// @Date:2022-05-17 19:33:44
// @Description: 获取国家tag，存到set里每次随机取完事
// @param ctx *gin.Context
//
func GetTTTag(ctx *gin.Context) {
	var r entity.GetTikTokVideoTagResponse
	country := ctx.Query("country")

	if country == "" {
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		prometheus.ReportKeeperResourceError("GetTTTag", "GetTTTag", "country err")
		return
	}
	CountryList := map[string]struct{}{
		"BR": {},
		"CO": {},
		"ID": {},
		"MX": {},
		"PE": {},
	}
	//不在这个列表里的统一变成US
	_, ok := CountryList[country]
	if country == "" || !ok {
		country = "US"
	}
	TTTagKey := keeper.GetCountryTagRK(country)
	//随机取4个tag吐出
	result, err := datamanager.GetRedisInstance().SRandMemberN(TTTagKey, 4)
	if err != nil {
		HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, nil)
		prometheus.ReportKeeperResourceError("GetTTTag", "GetTTTag", "get redis err")
		return
	}

	r.Result = result

	HttpSuccessResp(ctx, http.StatusOK, "success", r)
}

func GetTTByTag(ctx *gin.Context) {
	tagName := ctx.Query("tag")
	Country := ctx.Query("country")
	CountryList := map[string]struct{}{
		"BR": {},
		"CO": {},
		"ID": {},
		"MX": {},
		"PE": {},
	}
	//不在这个列表里的统一变成US
	_, ok := CountryList[Country]
	if Country == "" || !ok {
		Country = "US"
	}
	if tagName == "" {
		prometheus.ReportKeeperResourceError("GetTTByTag", "GetTTByTag", "tag err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamStructErrCode, "")
		return
	}

	var resp entity.GetTikTokResourceResponse

	result, err := datamanager.GetMysqlInstance().GetTikTokByVideoTagAndCountry(tagName, Country)
	if err != nil {
		logger.Logger.Error(fc.MysqlExecErrorMsg + " GetTikTokByVideoTagAndCountry:" + err.Error())
		prometheus.ReportKeeperResourceError("GetTTByTag", "GetTTByTag", "GetTikTokByVideoTagAndCountry error")
		HttpSuccessResp(ctx, http.StatusOK, "success", nil)
		return
	}

	//每次接口打乱顺序
	tempMap := make(map[string]entity.TikTokResourceInfo)
	for _, v := range result {
		if v.VideoID != "" {
			tempMap[v.VideoID] = v
		}
	}

	var ttri []entity.TikTokResourceInfo
	for _, v := range tempMap {
		ttri = append(ttri, v)
	}

	resp.Result = ttri
	HttpSuccessResp(ctx, http.StatusOK, "success", resp)
}

//
// OperateValueTag
// @Author:hhz
// @Date:2022-05-17 16:45:02
// @Description: 添加删除展示Tag
// @param ctx *gin.Context
//
func OperateValueTag(ctx *gin.Context) {
	option := ctx.Query("op")
	tag := ctx.Query("tag")
	if tag == "" || option == "" {
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}
	rk := fc.KeeperStatusValueTagsSet
	redisHandle := datamanager.GetRedisInstance()
	if option == "add" {
		_, err := redisHandle.SAdd(rk, tag)
		if err != nil {
			HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, err.Error())
			return
		}
	} else if option == "del" {
		_, err := redisHandle.SRem(rk, tag)
		if err != nil {
			HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, err.Error())
			return
		}
	} else {
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, "Invalid Option")
		return
	}
	HttpSuccessResp(ctx, http.StatusOK, "ok", nil)
}

//
// OperateTagHeadPhoto
// @Author:hhz
// @Date:2022-05-17 16:47:40
// @Description: 添加删除tag图标
// @param ctx *gin.Context
//
func OperateTagHeadPhoto(ctx *gin.Context) {
	option := ctx.Query("op")
	tag := ctx.Query("tag")
	photo := ctx.Query("photo")
	if tag == "" || option == "" || photo == "" {
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}
	rk := fmt.Sprintf(fc.KeeperStatusTagHeadPhotosSet, tag)
	redisHandle := datamanager.GetRedisInstance()
	if option == "add" {
		_, err := redisHandle.SAdd(rk, photo)
		if err != nil {
			HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, err.Error())
			return
		}
	} else if option == "del" {
		_, err := redisHandle.SRem(rk, photo)
		if err != nil {
			HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, err.Error())
			return
		}
	} else {
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, "Invalid Option")
		return
	}
	HttpSuccessResp(ctx, http.StatusOK, "ok", nil)
}

//
// GetIURTagList
// @Author:hhz
// @Date:2022-05-17 20:31:01
// @Description: Keeper status页面获取展示tag
// @param ctx *gin.Context
//
func GetIURTagList(ctx *gin.Context) {
	resp := make([]entity.KeeperStatusTag, 0)
	redisHandle := datamanager.GetRedisInstance()
	tagList, err := redisHandle.SMembers(fc.KeeperStatusValueTagsSet)
	if err == nil {
		for _, tag := range tagList {
			rk := fmt.Sprintf(fc.KeeperStatusTagHeadPhotosSet, tag)
			headPhoto, err := redisHandle.SRandMember(rk)
			if err == nil && headPhoto != "" {
				statusTag := entity.KeeperStatusTag{
					Name:      tag,
					HeadPhoto: headPhoto,
				}
				resp = append(resp, statusTag)
			}
		}
	}
	go keeper.GetLastFMResource()

	HttpSuccessResp(ctx, http.StatusOK, "success", resp)
}

//
// GetIURListByTag
// @Author:hhz
// @Date:2022-05-15 20:29:58
// @Description: 根据tag获取imgur素材列表
// @param ctx *gin.Context
//
func GetIURListByTag(ctx *gin.Context) {
	page, num := 1, 20
	tagName, tPage, tNum := ctx.Query("tag"), ctx.Query("page"), ctx.Query("num")
	if tagName == "" {
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamStructErrCode, "")
		return
	}
	if tPage != "" {
		page, _ = strconv.Atoi(tPage)
	}
	if tNum != "" {
		num, _ = strconv.Atoi(tNum)
	}
	redisHeper := datamanager.GetRedisInstance()
	rk := keeper.GetStatusTagFeedRK(tagName)
	nums, _ := redisHeper.ZCard(rk)

	resp := entity.GetImgUrResourceResponse{
		Nums: nums,
		List: []entity.KeeperStatusResourceData{},
	}
	jsonList, _ := redisHeper.ZRange(rk, int64((page-1)*num), int64(page*num-1))

	for _, v := range jsonList {
		tmp := &entity.KeeperStatusResourceData{}
		json.Unmarshal([]byte(v), &tmp)
		resp.List = append(resp.List, *tmp)
	}

	HttpSuccessResp(ctx, http.StatusOK, "success", resp)
}

func UpdateSource(ctx *gin.Context) {
	logger.Logger.Info("start getting new single music")
	go ytsource.GetNewSingleVideoInfo()

	logger.Logger.Info("start getting trending music")
	go ytsource.GetTrendingResource()

	logger.Logger.Info("start getting playlist music")
	go ytsource.GetPlaylistResource()

	logger.Logger.Info("start getting fyou music interesting music")
	go ytsource.GetForYouSource()

	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)
}

func makeSaveInfo(source string, r entity.RecordOperationRequest) string {
	//source empty indicate new data to save redis
	recordOPArray := strings.Split(source, fc.GapStr)
	if len(recordOPArray) == 12 {
		sc, _ := strconv.Atoi(recordOPArray[3])
		r.ShowCount += sc
		tc, _ := strconv.Atoi(recordOPArray[4])
		r.TapCount += tc
		wc, _ := strconv.Atoi(recordOPArray[5])
		r.WatchCount += wc

		if recordOPArray[7] != "" {
			r.VideoWaitTimeStr += "-" + recordOPArray[7]
		}

		swc, _ := strconv.Atoi(recordOPArray[8])
		r.SendWhatsappCount += swc
		rsc, _ := strconv.Atoi(recordOPArray[9])
		r.ShareCount += rsc
		dc, _ := strconv.Atoi(recordOPArray[10])
		r.DownloadCount += dc
		rc, _ := strconv.Atoi(recordOPArray[11])
		r.ReturnCount += rc
	}

	buff := bytes.Buffer{}
	buff.WriteString(r.UserID)
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.Source))
	buff.WriteString(fc.GapStr)
	buff.WriteString(r.VideoID)
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.ShowCount))
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.TapCount))
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.WatchCount))
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.IsCompleteShow))
	buff.WriteString(fc.GapStr)
	buff.WriteString(r.VideoWaitTimeStr)
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.SendWhatsappCount))
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.ShareCount))
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.DownloadCount))
	buff.WriteString(fc.GapStr)
	buff.WriteString(strconv.Itoa(r.ReturnCount))

	return buff.String()
}

func RecordOperation(ctx *gin.Context) {
	//record user operation info
	go saveUserOPOfDay()

	var r entity.RecordOperationRequest

	err := ctx.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		logger.Logger.Warn("ShouldBindWith err", err)
		prometheus.ReportKeeperResourceError("RecordOperation", "RecordOperation", "ShouldBindWith err")
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
		return
	}

	r.VideoWaitTimeStr = strconv.Itoa(r.VideoWaitTime)
	allOPHashKeys := utils.GetUserIDOfDay(0)
	redisHelper := datamanager.GetRedisInstance()
	redisHelper.ZAdd(allOPHashKeys, redis.Z{
		Score:  float64(time.Now().Unix() + int64(r.WatchCount)),
		Member: r.UserID,
	})

	userOPKeys := utils.GetUserOperationOfDay(r.UserID, 0)
	opInfo, err := redisHelper.HGet(userOPKeys, r.UserID)
	if err != nil {
		logger.Logger.Warn("redis HGet userOPKeys err", err, userOPKeys, r.UserID)
	}

	fmt.Println("allOPHashKeys:", allOPHashKeys)
	fmt.Println("userOPKeys:", userOPKeys)

	newOpInfo := makeSaveInfo(opInfo, r)
	if newOpInfo != "" {
		_, err = redisHelper.HSet(userOPKeys, r.UserID, newOpInfo)
		if err != nil {
			logger.Logger.Warn("redis HSet userOPKeys err", err, userOPKeys, r.VideoID, newOpInfo)
			prometheus.ReportKeeperResourceError("RecordOperation", "RecordOperation", "redis err")
			HttpFailResp(ctx, http.StatusOK, fc.RedisServerExecErrorCode, nil)
			return
		}
	}

	redisHelper.Expire(allOPHashKeys, 48*time.Hour)
	redisHelper.Expire(userOPKeys, 48*time.Hour)

	HttpSuccessResp(ctx, http.StatusOK, "success", nil)
}

func saveUserOPOfDay() {
	//the seconds to second day 1 am
	ts := time.Now().AddDate(0, 0, 1)
	expire := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location()).Unix()
	expire = expire - time.Now().Unix() + 3600

	dataTime := time.Now().Format("2006-01-02")
	key := "RECORD_USER_OPERATION_KEY:" + dataTime

	redisHelper := datamanager.GetRedisInstance()
	flag, _ := redisHelper.SetNX(key, 1, time.Duration(expire)*time.Second)
	if !flag {
		logger.Logger.Info("DOWNLOAD_YOUTUBE_INFO_KEY is set not need download")
		return
	}

	timestamp := time.Now().Unix()
	allOPHashKeys := utils.GetUserIDOfDay(-1)

	again := true
	index := 0
	start := 0
	for again {
		start = index * 50
		end := start + 50
		fmt.Println(start, "-------", end, allOPHashKeys)
		userIDArray, err := redisHelper.ZRevRange(allOPHashKeys, int64(start), int64(end))
		if err != nil {
			logger.Logger.Warn("redis ZRevRange allOPHashKeys err", err, allOPHashKeys)
			return
		}
		fmt.Println("userIDArray-------", userIDArray)

		if len(userIDArray) == 0 {
			logger.Logger.Warn("redis ZRevRange allOPHashKeys empty data")
			return
		}

		for _, userID := range userIDArray {
			time.Sleep(500 * time.Millisecond)
			if userID == "" {
				continue
			}

			userOPKeys := utils.GetUserOperationOfDay(userID, -1)
			recordInfoStrMap, err := redisHelper.HGetAll(userOPKeys)
			if err != nil {
				logger.Logger.Warn("redis HGetAll userOPKeys err", err, userOPKeys)
				continue
			}
			fmt.Println("recordInfoStrMap-------", recordInfoStrMap)

			var result []entity.UserOperationRecordInfo
			for _, recordInfoStr := range recordInfoStrMap {
				recordOPArray := strings.Split(recordInfoStr, fc.GapStr)
				if len(recordOPArray) != 12 {
					continue
				}
				fmt.Println("recordOPArray[0]-------", recordOPArray[0])

				ui := recordOPArray[0]
				source, _ := strconv.Atoi(recordOPArray[1])
				vi := recordOPArray[2]
				sc, _ := strconv.Atoi(recordOPArray[3])
				tc, _ := strconv.Atoi(recordOPArray[4])
				wc, _ := strconv.Atoi(recordOPArray[5])
				ics, _ := strconv.Atoi(recordOPArray[6])
				swc, _ := strconv.Atoi(recordOPArray[8])
				share, _ := strconv.Atoi(recordOPArray[9])
				dc, _ := strconv.Atoi(recordOPArray[10])
				rc, _ := strconv.Atoi(recordOPArray[11])

				if ui == "" || vi == "" {
					continue
				}

				result = append(result, entity.UserOperationRecordInfo{
					UserID:            ui,
					Source:            source,
					VideoID:           vi,
					ShowCount:         sc,
					TapCount:          tc,
					WatchCount:        wc,
					IsCompleteShow:    ics,
					VideoWaitTime:     utils.GetMaxNumFromStr(recordOPArray[7]),
					SendWhatsappCount: swc,
					ShareCount:        share,
					DownloadCount:     dc,
					ReturnCount:       rc,
					DataTime:          dataTime,
					CreateTime:        timestamp,
				})
			}

			err = datamanager.GetMysqlInstance().InsertUserOperationRecordInfo(result)
			if err != nil {
				logger.Logger.Warn("InsertUserOperationRecordInfo err", err)
			}

		}

		index += 1
	}

}

func LoadMusicPageInfo(ctx *gin.Context) {
	//Todo:hitsongs1
	//go ytsource.LoadHitSongsPageInfoToFeishu()
	go ytsource.NewLoadHitSongsPageInfo()
	//Todo:hitsongs2

	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)

}

func LoadYtMusicPageInfo(ctx *gin.Context) {
	go ytsource.LoadYtHitSongsPageInfo()

	HttpSuccessResp(ctx, http.StatusOK, "exec success", nil)

}

func GetMusicPageInfo(ctx *gin.Context) {

	pageStr := ctx.Query("page")
	pageSizeStr := ctx.Query("page_size")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		logger.Logger.Error("request param err:", err)
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		logger.Logger.Error("request param err:", err)
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
	}

	var data []entity.MusicPageInfo

	mysqlHelper := datamanager.GetMysqlInstance()
	data, err = mysqlHelper.GetMusicpageInfoByIdRange(page, pageSize)
	if err != nil {
		logger.Logger.Error("get data err:", err)
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
	}
	HttpSuccessResp(ctx, http.StatusOK, "exec success", data)
}

func GetYtMusicPageInfo(ctx *gin.Context) {

	pageStr := ctx.Query("page")
	pageSizeStr := ctx.Query("page_size")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		logger.Logger.Error("request param err:", err)
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		logger.Logger.Error("request param err:", err)
		HttpFailResp(ctx, http.StatusOK, fc.RequestParamErrCode, nil)
	}

	var data []entity.MusicPageInfo

	mysqlHelper := datamanager.GetMysqlInstance()
	data, err = mysqlHelper.GetYtMusicpageInfoByIdRange(page, pageSize)
	if err != nil {
		logger.Logger.Error("get data err:", err)
		HttpFailResp(ctx, http.StatusOK, fc.MysqlExecErrorCode, nil)
	}
	HttpSuccessResp(ctx, http.StatusOK, "exec success", data)
}

func Transform(c *gin.Context) {

	fileHeader, err := c.FormFile("file")
	if err != nil {
		HttpFailResp(c, http.StatusOK, fc.RequestFormDataErrCode, nil)
		return
	}

	var fileType sticker.ImgType

	switch fileHeader.Header.Get("Content-Type") {
	case "image/gif":
		fileType = sticker.GIF
	case "image/jpeg":
		fileType = sticker.JPEG
	case "image/png":
		fileType = sticker.PNG
	default:
		{
			HttpFailResp(c, http.StatusOK, fc.RequestFormDataErrCode, nil)
			return
		}
	}

	file, err := fileHeader.Open()
	if err != nil {
		HttpFailResp(c, http.StatusOK, fc.RequestFormDataErrCode, nil)
		return
	}

	source, err := ioutil.ReadAll(file)

	if err != nil {
		HttpFailResp(c, http.StatusOK, fc.RequestFormDataErrCode, nil)
		return
	}

	_, targetFile, err := sticker.Transform(source, fileType)
	if err != nil {
		HttpFailResp(c, http.StatusOK, fc.TransformFailCode, nil)
		return
	}

	fmt.Println(targetFile)
	c.File(targetFile)
	defer func() {
		os.Remove(targetFile)
	}()

}
