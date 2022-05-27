package logger

import (
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

//全局logger对象
var Logger = logrus.New()
var once sync.Once

func Init() *logrus.Logger {
	once.Do(func() {

		Logger.Out, _ = os.OpenFile("my.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)

		//设置日志级别
		Logger.SetLevel(logrus.DebugLevel)

		//设置日志格式
		Logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	})
	return Logger
}
