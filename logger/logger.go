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
		//now := time.Now()
		//
		//logFilePath := ""
		//if dir, err := os.Getwd(); err == nil {
		//	logFilePath = dir + "/logs/"
		//}
		//
		//if err := os.MkdirAll(logFilePath, 0777); err != nil {
		//	fmt.Println(err.Error())
		//}
		//
		//logFileName := now.Format("2006-01-02") + ".log"
		//
		//fileName := path.Join(logFilePath, logFileName)
		//if _, err := os.Stat(fileName); err != nil {
		//	if _, err := os.Create(fileName); err != nil {
		//		fmt.Println(err.Error())
		//	}
		//}
		//
		//src, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		//if err != nil {
		//	fmt.Println("err", err)
		//}

		//设置输出
		//Logger.Out = os.Stdout

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
