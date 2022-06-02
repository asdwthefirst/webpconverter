package apiserver

import (
	"fmt"
	fc "webpconverter/feedconst"
	"webpconverter/sticker"

	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
)

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
	defer file.Close()

	source, err := ioutil.ReadAll(file)
	//fmt.Println("source len", len(source))

	if err != nil {
		HttpFailResp(c, http.StatusOK, fc.RequestFormDataErrCode, nil)
		return
	}

	_, targetFile, err := sticker.Transform(source, fileType)
	fmt.Println(targetFile)
	if err != nil {
		HttpFailResp(c, http.StatusOK, fc.TransformFailCode, nil)
		return
	}

	c.File(targetFile)
	//defer func() {
	//	os.Remove(targetFile)
	//}()

}
