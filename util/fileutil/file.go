package fileutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type File struct {
	Name    string
	Path    string
	ModTime time.Time
	Size    int64
	IsDir   bool
}

type Files []*File

const (
	PathSep = string(os.PathSeparator)
)

// GetAllFiles 获取指定目录下的所有文件,包含子目录下的文件
func GetAllFiles(dirPath string) (files Files, err error) {
	fileInfos, err := getFiles(dirPath)
	if err != nil {
		return nil, err
	}

	for _, fi := range fileInfos {
		fullPath := dirPath + PathSep + fi.Name()
		if fi.IsDir() { // 目录, 递归遍历
			fs, err := GetAllFiles(fullPath)
			if err != nil {
				return nil, err
			}
			files = append(files, fs...)
		} else {
			files = append(files, &File{Name: fi.Name(), Path: fullPath, ModTime: fi.ModTime(), Size: fi.Size()})
		}
	}

	return files, nil
}

// GetFiles 获取指定目录下的所有文件,不包含子目录下的文件
func GetFiles(dirPath string) (files Files, err error) {
	fileInfos, err := getFiles(dirPath)
	if err != nil {
		return nil, err
	}

	for _, fi := range fileInfos {
		if !fi.IsDir() {
			files = append(files, &File{Name: fi.Name(), Path: dirPath + PathSep + fi.Name(), ModTime: fi.ModTime(), Size: fi.Size()})
		}
	}

	return files, nil
}

func GetDirs(dirPath string, childDir ...bool) (files Files, err error) {
	fileInfos, err := getFiles(dirPath)
	if err != nil {
		return nil, err
	}

	for _, fi := range fileInfos {
		if fi.IsDir() {
			fullPath := dirPath + PathSep + fi.Name()
			files = append(files, &File{Name: fi.Name(), Path: fullPath, ModTime: fi.ModTime(), Size: fi.Size()})
			if len(childDir) > 0 && childDir[0] {
				fs, err := GetDirs(fullPath, true)
				if err != nil {
					return nil, err
				}
				files = append(files, fs...)
			}
		}
	}

	return files, nil
}

// ReadFile 读取文件内容
func ReadFile(path string) (string, error) {
	fi, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fi.Close()
	fd, err := ioutil.ReadAll(fi)
	return string(fd), nil
}

// CreateFile 创建文件
func CreateFile(path string, contents string, override ...bool) error {
	dir := ""
	if idx := strings.LastIndex(path, "/"); idx > -1 {
		dir = path[0:strings.LastIndex(path, "/")]
	}

	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	mod := os.O_CREATE | os.O_EXCL | os.O_WRONLY
	if len(override) > 0 && override[0] {
		mod = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	file, err := os.OpenFile(path, mod, 0755)

	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.Write([]byte(contents))

	if err != nil {
		return err
	}

	return nil
}

// DeleteFile 删除文件
func DeleteFile(path string) error {
	return os.Remove(path)
}

// RenameFile 重命名文件
func RenameFile(path string, newNname string, override ...bool) error {
	if len(override) == 0 || !override[0] {
		if _, err := os.Stat(newNname); err == nil {
			return fmt.Errorf("File %v exists", newNname)
		}
	}
	return os.Rename(path, newNname)
}

// WriteFile 写入文件
func WriteFile(path string, contents string, override ...bool) error {
	mod := os.O_WRONLY
	if len(override) > 0 && override[0] {
		mod = os.O_WRONLY | os.O_TRUNC
	}
	file, err := os.OpenFile(path, mod, 0755)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.Write([]byte(contents))

	if err != nil {
		return err
	}

	return nil
}

// MakeDir 创建目录
func MakeDir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

func Combine(basePath string, pathNames ...string) string {
	p := basePath
	for _, path := range pathNames {
		if strings.HasSuffix(p, PathSep) || strings.HasPrefix(path, PathSep) {
			p += path
		} else {
			p += PathSep + path
		}
	}

	return p
}

func getFiles(dirPath string) ([]os.FileInfo, error) {
	fileInfos, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	return fileInfos, nil
}

func (this Files) Len() int {
	return len(this)
}

func (this Files) Less(i, j int) bool {
	return this[i].ModTime.Before(this[j].ModTime)
}

func (this Files) Swap(i, j int) {
	var temp *File = this[i]
	this[i] = this[j]
	this[j] = temp
}
