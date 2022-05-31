package sticker

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"webpconverter/logger"
)

type ImgType int

const (
	GIF  = 1
	JPEG = 2
	PNG  = 3
)

type transformer func([]byte) ([]byte, string, error)

var transformerMap = map[ImgType]transformer{
	GIF:  GIF2WebpTransformer,
	JPEG: JPEG2WebpTransformer,
	PNG:  PNG2WebpTransformer,
}

var sourceFilePath = "../imgsource"
var targetFilePath = "../imgtarget"

func Transform(source []byte, imgType ImgType) (target []byte, targetFile string, err error) {

	if len(source) == 0 {
		err = errors.New("empty img source")
		logger.Logger.Error(err)
		return nil, "", err
	}

	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		// 必须分成两步：先创建文件夹、再修改权限
		os.Mkdir(sourceFilePath, 0777) //0777也可以os.ModePerm
		os.Chmod(sourceFilePath, 0777)
	}
	if _, err := os.Stat(targetFilePath); os.IsNotExist(err) {
		// 必须分成两步：先创建文件夹、再修改权限
		os.Mkdir(targetFilePath, 0777) //0777也可以os.ModePerm
		os.Chmod(targetFilePath, 0777)
	}

	switch imgType {
	case GIF:
		return transformerMap[GIF](source)
	case JPEG:
		return transformerMap[JPEG](source)
	case PNG:
		return transformerMap[PNG](source)
	default:
		return transformerMap[GIF](source)
	}

}

func GIF2WebpTransformer(source []byte) (target []byte, targetFile string, err error) {

	uid := uuid.New()
	sourceFile := path.Join(sourceFilePath, fmt.Sprintf("%v.gif", uid))
	targetFile = path.Join(targetFilePath, fmt.Sprintf("%v.webp", uid))

	ioutil.WriteFile(sourceFile, source, 0777)
	if err != nil {
		logger.Logger.Error("gif2webp writefile err:", err)
		return
	}

	cmd := exec.Command("gif2webp", sourceFile, "-o", targetFile)
	//cmd.Run()
	output, _ := cmd.Output()
	logger.Logger.Info(cmd.String(), "----out:", output)

	//defer os.Remove(sourceFile)

	target, err = ioutil.ReadFile(targetFile)
	if err != nil {
		logger.Logger.Error("gif2webp readfile err:", err)
		return
	}

	return
}

func JPEG2WebpTransformer(source []byte) (target []byte, targetFile string, err error) {

	//var sourceBuf bytes.Buffer

	uid := uuid.New()
	sourceFile := path.Join(sourceFilePath, fmt.Sprintf("%v.jpg", uid))
	targetFile = path.Join(targetFilePath, fmt.Sprintf("%v.webp", uid))

	ioutil.WriteFile(sourceFile, source, 0777)
	if err != nil {
		logger.Logger.Error("jpeg2webp writefile err:", err)
		return
	}

	cmd := exec.Command("img2webp", "-lossy", sourceFile, "-o", targetFile)
	//cmd.Run()
	output, _ := cmd.Output()
	logger.Logger.Info(cmd.String(), "----out:", output)

	//defer os.Remove(sourceFile)

	target, err = ioutil.ReadFile(targetFile)
	if err != nil {
		logger.Logger.Error("jpeg2webp readfile err:", err)
		return
	}

	return
}
func PNG2WebpTransformer(source []byte) (target []byte, targetFile string, err error) {

	uid := uuid.New()
	sourceFile := path.Join(sourceFilePath, fmt.Sprintf("%v.png", uid))
	targetFile = path.Join(targetFilePath, fmt.Sprintf("%v.webp", uid))

	err = ioutil.WriteFile(sourceFile, source, 0777)
	if err != nil {
		logger.Logger.Error("png2webp writefile err:", err)
		return
	}

	cmd := exec.Command("img2webp", "-lossy", sourceFile, "-o", targetFile)
	//cmd.Run()
	output, _ := cmd.Output()
	logger.Logger.Info(cmd.String(), "----out:", output)

	//defer os.Remove(sourceFile)

	target, err = ioutil.ReadFile(targetFile)
	if err != nil {
		logger.Logger.Error("png2webp readfile err:", err)
		return
	}

	return
}
