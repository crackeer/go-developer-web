package fsutil

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateTarFile ...
//
//	@param srcDir
//	@param dest
//	@return error
func CreateTarFile(srcDir string, dest string) error {

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	var tarWriter *tar.Writer = tar.NewWriter(destFile)
	defer tarWriter.Close()

	fileMap := GetFilesMap(srcDir)

	return TarWriteFiles(tarWriter, fileMap, "")
}

// TarWriteFile ...
//
//	@param tw
//	@param tarFilePath
//	@param localFilePath
//	@return error
func TarWriteFile(tw *tar.Writer, tarFilePath, localFilePath string) error {

	if tw == nil {
		return fmt.Errorf("tar writer is nil")
	}

	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("open file `%s` error: %s", localFilePath, err.Error())
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("get file stat `%s` error: %s", localFilePath, err.Error())
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("tar.FileInfoHeader `%s` error: %s", localFilePath, err.Error())
	}
	header.Name = tarFilePath

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("tar.WriteHeader `%s` error: %s", localFilePath, err.Error())
	}

	if _, err = io.Copy(tw, file); err != nil {
		return fmt.Errorf("copy file `%s` error: %s", localFilePath, err.Error())
	}
	return nil
}

// TarWriteFiles ...
//
//	@param tw
//	@param fileMap
//	@param prefix
//	@return error
func TarWriteFiles(tw *tar.Writer, fileMap map[string]string, prefix string) error {
	for key, value := range fileMap {
		if err := TarWriteFile(tw, prefix+key, value); err != nil {
			return err
		}
	}
	return nil
}

// ExtractTarFile
//
//	@param tarFile
//	@param destDir
//	@return error
func ExtractTarFile(tarFile string, destDir string) error {

	var (
		err       error
		osFile    *os.File
		tarReader *tar.Reader
	)

	osFile, err = os.Open(tarFile)
	if err != nil {
		return fmt.Errorf("open %s error: %s", tarFile, err.Error())
	}
	defer osFile.Close()
	tarReader = tar.NewReader(osFile)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return fmt.Errorf("error occur:%s", err.Error())
			}
		}

		curFile := header.FileInfo()
		if curFile.IsDir() {
			continue
		}

		tmpFilePath := filepath.Join(destDir, header.Name)
		dir, _ := filepath.Split(tmpFilePath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("mkdir `%s` error: %s", dir, err.Error())
			}
		}

		tmpFile, err := os.Create(tmpFilePath)
		if err != nil {
			return fmt.Errorf("create file %s error: %s", tmpFilePath, err.Error())
		}
		io.Copy(tmpFile, tarReader)
		tmpFile.Close()
	}

	return nil
}
