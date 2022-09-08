package tool

import "os"

// CheckAndCreateDir 检查有没有文件夹，没有则创建
func CheckAndCreateDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			panic(err)
		}
	}
}
