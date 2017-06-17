// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of msg-net 
// 
// The msg-net is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// The msg-net is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// 
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

//Package logger 提供记录日记文件的函数
//  依赖情况:
//          "github.com/Sirupsen/logrus"
package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/bocheninc/msg-net/config"
	"github.com/bocheninc/msg-net/util"
)

var logger *logrus.Logger
var day string

func init() {
	Init()
}

//Init Initialization
func Init() {
	logger = logrus.New()
	if level := config.GetString("logger.level"); level != "" {
		if lv, err := logrus.ParseLevel(level); err == nil {
			logger.Level = lv
		} else {
			Errorf("unsupport logger.level %s --- %v\n", level, err)
		}
	}

}

//SetOut sets out log
func SetOut() {
	if out := config.GetString("logger.out"); out != "" {
		if day == "" {
			if !util.IsDirExist(out) {
				if err := util.MkDir(out); err != nil {
					Errorf("failed to open file %s for logger.Out --- %v", out, err)
					return
				}
			}
		}

		day2 := time.Now().Format("2006-01-02")
		if day == "" || day != day2 {
			switch logger.Out.(type) {
			case *os.File:
				if logger.Out != os.Stderr && logger.Out != os.Stdout {
					logger.Out.(*os.File).Close()
				}
			}
			day = day2
			var fileName string
			if config.GetString("router.id") == "" {
				fileName = filepath.Join(out, day+"_"+strings.Split(config.GetString("router.address"), ":")[1]+".log")
			} else {
				fileName = filepath.Join(out, day+"_"+config.GetString("router.id")+".log")
			}
			if f, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666); err == nil {
				logger.Out = f
			} else {
				logger.Out = os.Stderr
				Errorf("failed to open file %s for logger.Out --- %v", fileName, err)
			}
		}
	}

	if formatter := config.GetString("logger.formatter"); formatter != "" {
		switch f := strings.ToLower(formatter); f {
		case "json":
			logger.Formatter = &logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			}
		case "text":
			logger.Formatter = &logrus.TextFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
				FullTimestamp:   true,
			}
		default:
			Errorf("unsupport logger.formatter %s\n", formatter)
		}
	}
}

//String returns summary
func String() string {
	formatter := "unkown"
	switch logger.Formatter.(type) {
	case *logrus.JSONFormatter:
		formatter = "json"
	case *logrus.TextFormatter:
		formatter = "text"
	}
	out := "unkown"
	switch logger.Out.(type) {
	case *os.File:
		out = logger.Out.(*os.File).Name()
	}

	m := make(map[string]interface{})
	m["formatter"] = formatter
	m["level"] = logger.Level.String()
	m["out"] = out

	bytes, _ := json.Marshal(m)
	return string(bytes)
}

//Debug logs a message at level Debug on the standard logger
func Debug(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Debug(args...)
}

//Info logs a message at level Info on the standard logger
func Info(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Info(args...)
}

//Warn logs a message at level Warn on the standard logger
func Warn(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Warn(args...)
}

//Error logs a message at level Error on the standard logger
func Error(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Error(args...)
}

//Panic logs a message at level Panic on the standard logger
func Panic(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Panic(args...)
}

//Fatal logs a message at level Fatal on the standard logger
func Fatal(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Fatal(args...)
}

//Debugln logs a message at level Debug on the standard logger
func Debugln(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Debugln(args...)
}

//Infoln logs a message at level Info on the standard logger
func Infoln(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Infoln(args...)
}

//Warnln logs a message at level Warn on the standard logger
func Warnln(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Warnln(args...)
}

//Errorln logs a message at level Error on the standard logger
func Errorln(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Errorln(args...)
}

//Panicln logs a message at level Panic on the standard logger
func Panicln(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Panicln(args...)
}

//Fatalln logs a message at level Fatal on the standard logger
func Fatalln(args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Fatalln(args...)
}

//Debugf logs a message at level Debug on the standard logger
func Debugf(format string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Debugf(format, args...)
}

//Infof logs a message at level Info on the standard logger
func Infof(format string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Infof(format, args...)
}

//Warnf logs a message at level Warn on the standard logger
func Warnf(format string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Warnf(format, args...)
}

//Errorf logs a message at level Error on the standard logger
func Errorf(format string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Errorf(format, args...)
}

//Panicf logs a message at level Panic on the standard logger
func Panicf(format string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Panicf(format, args...)
}

//Fatalf logs a message at level Fatal on the standard logger
func Fatalf(format string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"location": util.CallerInfo(2),
	}).Fatalf(format, args...)
}
