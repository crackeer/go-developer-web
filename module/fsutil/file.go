package fsutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mholt/archiver/v4"
	"github.com/tidwall/gjson"
)

func GetNowTimeStamp() string {
	return time.Now().Format("2006-01-02-15-04-05")
}

// MakeVarLogFile
//
//	@param tag
//	@return string
func MakeVarLogFile(tag string) (string, error) {
	logFile := filepath.Join("/var/log", tag, GetNowTimeStamp()+".log")
	dir, _ := filepath.Split(logFile)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}
	return logFile, nil
}

// MakeVarLogDir MakeVarLog
//
//	@param tag
//	@return string
func MakeVarLogDir(tag string) string {
	workDir := filepath.Join("/var/log", tag, GetNowTimeStamp())
	os.MkdirAll(workDir, os.ModePerm)
	return workDir
}

// MakeTmpWorkDir
//
//	@param tag
//	@return string
func MakeTmpWorkDir(tag string) string {
	workDir := filepath.Join("/tmp", tag, GetNowTimeStamp())
	os.MkdirAll(workDir, os.ModePerm)
	return workDir
}

// GetFiles
//
//	@param folder
//	@return []string
func GetFiles(folder string) []string {
	files, _ := os.ReadDir(folder)
	retData := []string{}
	for _, file := range files {
		if file.IsDir() {
			tmp := GetFiles(filepath.Join(folder, file.Name()))
			retData = append(retData, tmp...)
		} else {
			retData = append(retData, filepath.Join(folder, file.Name()))
		}
	}
	return retData
}

// GetFilesMap
//
//	@param fileDir
//	@return map
func GetFilesMap(folder string) map[string]string {
	fileList := GetFiles(folder)
	if len(fileList) < 1 {
		return nil
	}
	fileMap := map[string]string{}
	for _, file := range fileList {
		key := file[len(folder)+1:]
		fileMap[key] = file
	}
	return fileMap
}

// ReadConfig
//
//	@param dest
//	@return error
func ReadConfig(dest interface{}) error {
	if len(os.Args) < 2 {
		return errors.New("no config file")
	}
	bytes, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dest)
}

func GetDirFilesAsMap(fileDir string) map[string]string {
	fileList := GetFiles(fileDir)
	if len(fileList) < 1 {
		return nil
	}
	fileMap := map[string]string{}
	for _, file := range fileList {
		key := file[len(fileDir)+1:]
		fileMap[key] = file
	}
	return fileMap
}

// Zip
//
//	@param srcDir
//	@param dest
//	@return error
func Zip(srcDir, dest string) error {
	file, err := os.Open(srcDir)
	if err != nil {
		return err
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("file stat error: %v", err)
	}

	fileMap := map[string]string{}
	if fileStat.IsDir() {
		fileMap = GetDirFilesAsMap(srcDir)
	} else {
		// srcDir是一个文件
		_, name := filepath.Split(srcDir)
		fileMap[name] = srcDir
	}

	fileMapRevert := map[string]string{}

	for key, value := range fileMap {
		fileMapRevert[value] = key
	}

	// map files on disk to their paths in the archive
	files, err := archiver.FilesFromDisk(nil, fileMapRevert)
	if err != nil {
		return err
	}

	// create the output file we'll write to
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// we can use the CompressedArchive type to gzip a tarball
	// (compression is not required; you could use Tar directly)
	format := archiver.CompressedArchive{
		Compression: nil,
		Archival:    archiver.Zip{},
	}

	// create the archive
	err = format.Archive(context.Background(), out, files)
	if err != nil {
		return err
	}
	return nil
}

// Unzip ...
//
//	@param srcFile
//	@param destSrc
//	@return error
func Unzip(srcFile, destSrc string) error {
	file, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer file.Close()

	format := &archiver.Zip{}
	return format.Extract(context.Background(), file, nil, func(ctx context.Context, f archiver.File) error {
		if f.IsDir() {
			return os.MkdirAll(filepath.Join(destSrc, f.NameInArchive), os.ModePerm)
		}
		reader, err := f.Open()
		if err != nil {
			return err
		}
		bytes, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("read %s error:%s", f.NameInArchive, err.Error())
		}

		tmpDest := filepath.Join(destSrc, f.NameInArchive)
		tmpDir := filepath.Dir(tmpDest)
		if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
			if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
				return err
			}
		}
		return os.WriteFile(filepath.Join(destSrc, f.NameInArchive), bytes, os.ModePerm)
	})
}

func JsonDecodeX(inputJSON string) (interface{}, error) {
	if len(inputJSON) < 1 {
		return nil, errors.New("cant decode empty string")
	}
	var data interface{}
	err := json.Unmarshal([]byte(inputJSON), &data)
	return data, err
}

// ExtractURL
//
//	@param value
//	@param matchFunc
//	@return []string
func ExtractURL(value interface{}, matchFunc func(string) bool) []string {
	if strValue, ok := value.(string); ok {
		if gjson.Valid(strValue) {
			var data interface{}
			if err := json.Unmarshal([]byte(strValue), &data); err == nil {
				return ExtractURL(data, matchFunc)
			}
		}
		if matchFunc(strValue) {
			return strings.Split(strValue, ",")
		}
		return []string{}
	}

	var retData []string
	if mapValue, ok := value.(map[string]interface{}); ok {
		for _, value := range mapValue {
			retData = append(retData, ExtractURL(value, matchFunc)...)
		}
		return retData
	}

	if listValue, ok := value.([]interface{}); ok {
		for _, value := range listValue {
			retData = append(retData, ExtractURL(value, matchFunc)...)
		}
		return retData
	}
	return nil
}

// ReadFileAs
//
//	@param path
//	@param v
//	@return error
func ReadFileAs(path string, v interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, v)
}

func UnzipFrom(srcFile io.Reader, destSrc string) error {

	format := &archiver.Zip{}
	return format.Extract(context.Background(), srcFile, nil, func(ctx context.Context, f archiver.File) error {
		if f.IsDir() {
			return os.MkdirAll(filepath.Join(destSrc, f.NameInArchive), os.ModePerm)
		}
		reader, err := f.Open()
		if err != nil {
			return err
		}
		bytes, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("read %s error:%s", f.NameInArchive, err.Error())
		}

		tmpDest := filepath.Join(destSrc, f.NameInArchive)
		tmpDir := filepath.Dir(tmpDest)
		if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
			if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
				return err
			}
		}
		return os.WriteFile(filepath.Join(destSrc, f.NameInArchive), bytes, os.ModePerm)
	})
}

func GetFileMapping(path string) (map[string]string, error) {
	retData := map[string]string{}
	err := filepath.Walk(path, func(tmpPath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Clean(path) == filepath.Clean(tmpPath) {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(path, tmpPath)
		if err != nil {
			return err
		}
		retData[relPath] = tmpPath
		return nil
	})

	return retData, err
}

func ExtractURLToMap(input interface{}, matchFunc func(string) bool, prefix string) map[string]string {
	retData := map[string]string{}
	if value, ok := input.(map[string]interface{}); ok {
		for key, tempValue := range value {
			if strValue, ok := tempValue.(string); ok {
				if matchFunc(strValue) {
					if len(prefix) > 0 {
						retData[prefix+"."+key] = strValue
					} else {
						retData[key] = strValue
					}
				}
			}
			subMap := ExtractURLToMap(tempValue, matchFunc, key)
			for subKey, subValue := range subMap {
				retData[subKey] = subValue
			}
		}
	}
	if listValue, ok := input.([]interface{}); ok {
		for index, tempValue := range listValue {
			if len(prefix) > 0 {
				subMap := ExtractURLToMap(tempValue, matchFunc, fmt.Sprintf("%s.%d", prefix, index))
				for subKey, subValue := range subMap {
					retData[subKey] = subValue
				}
			}
		}
	}
	return retData
}
