package tarpack

import (
	"archive/tar"
	//"compress/gzip"
	//"errors"
	"bytes"
	"fmt"
	"io/ioutil"
	//"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FilePredicate func(os.FileInfo) bool

// Limit filter scope to only files
func OnFiles(filters ...FilePredicate) FilePredicate {
	return func(file os.FileInfo) bool {
		if file.IsDir() {
			return true
		}
		for _, filter := range filters {
			if !filter(file) {
				return false
			}
		}
		return true
	}
}

// Limit filter scope to only dirs
func OnDirs(files []os.FileInfo, filters ...FilePredicate) FilePredicate {
	return func(file os.FileInfo) bool {
		if !file.IsDir() {
			return true
		}
		for _, filter := range filters {
			if !filter(file) {
				return false
			}
		}
		return true
	}
}

// Make sure none of the contained filters matches
func Not(filters ...FilePredicate) FilePredicate {
	return func(file os.FileInfo) bool {
		for _, filter := range filters {
			if filter(file) {
				return false
			}
		}
		return true
	}
}

func NameContainsAnyPredicate(whitelist ...string) FilePredicate {
	return func(file os.FileInfo) bool {
		for _, entry := range whitelist {
			if strings.Contains(file.Name, entry) {
				return true
			}
		}
		return false
	}
}

func PathMatchAnyPredicate(whitelist ...string) FilePredicate {
	return func(file os.FileInfo) bool {
		for _, entry := range whitelist {
			if match, err := filepath.Match(entry, name); match && err != nil {
				return true
			}
		}
		return false
	}
}

func Where(files []os.FileInfo, filters ...FilePredicate) ([]os.FileInfo, error) {
	result := make([]os.FileInfo, 0, len(files))
	for _, file := range files {
		for _, filter := range filters {
			if !filter(file) {
				continue
			}
			result = append(result, file)
		}
	}
	return result
}

/*
func DirFilter(whitelist, blacklist []string) func(string) bool {
	wLen := len(whitelist)
	bLen := len(blacklist)
	if wLen == 0 && bLen == 0 {
		return func(fileName string) bool { return true }
	}
	if wLen == 0 {
		return func(fileName string) bool { return !ContainsPredicate(blacklist, fileName) }
	}
	if bLen == 0 {
		return func(fileName string) bool { return ContainsPredicate(whitelist, fileName) }
	}
	return func(fileName string) bool {
		return ContainsPredicate(whitelist, fileName) && !ContainsPredicate(blacklist, fileName)
	}
}

func FileFilter(whitelist, blacklist []string) func(string) bool {
	wLen := len(whitelist)
	bLen := len(blacklist)
	if wLen == 0 && bLen == 0 {
		return func(fileName string) bool { return true }
	}
	if wLen == 0 {
		return func(fileName string) bool { return !FileMatchPredicate(blacklist, fileName) }
	}
	if bLen == 0 {
		return func(fileName string) bool { return FileMatchPredicate(whitelist, fileName) }
	}
	return func(fileName string) bool {
		return FileMatchPredicate(whitelist, fileName) && !FileMatchPredicate(blacklist, fileName)
	}
}
*/

func TarDir(rootPath, relPath string, filters ...func(os.FileInfo) bool) error {
	buffer := new(bytes.Buffer)
	handle := tar.NewWriter(buffer)
	return tarDir(handle, rootPath, relPath, folderFilter, fileFilter)
}

func tarDir(handle *tar.Writer, rootPath, relPath string, filters ...func(os.FileInfo) bool) error {
	fmt.Printf("Tarring dir: %s => %s\n", rootPath, relPath)
	var err error
	var files []os.FileInfo
	if files, err = ioutil.ReadDir(rootPath); err != nil {
		return err
	}
	for _, file := range Where(files, filters...) {
		newPath := relPath
		if len(relPath) != 0 && relPath[:len(relPath)] != "/" {
			newPath += "/"
		}
		/*
			for _, filter := range filters {
				if !filter(file) {
					continue
				}
			}
		*/
		if file.IsDir() {
			tarDir(handle, rootPath+"/"+file.Name(), newPath+file.Name(), folderFilter, fileFilter)
		} else { // File is a file
			fmt.Printf("Writing file: %s -> %s\n", relPath, file.Name())
			if err = writeFile(handle, rootPath+"/"+file.Name(), newPath+file.Name()); err != nil {
				fmt.Printf("Error writing file: %s\n", err)
				return err
			}
		}
	}
	return nil
}

/*
	if file.IsDir() { // File is a dir and not a file
		for _, filter := range filters {

			if !filter(file) {
				continue
			}
		}
		if !folderFilter(file.Name()) {
			continue
		}
		tarDir(handle, rootPath+"/"+file.Name(), newPath+file.Name(), folderFilter, fileFilter)
	} else { // File is a file and not a dir
		if _, filter := range filters {
			if !filter(file) {
				continue
			}
		}

		if !fileFilter(file.Name()) {
			continue
		}

		fmt.Printf("Writing file: %s -> %s\n", relPath, file.Name())
		if err = writeFile(handle, rootPath+"/"+file.Name(), newPath+file.Name()); err != nil {
			fmt.Printf("Error writing file: %s\n", err)
			return err
		}
	}
*/
//}
//return nil
//}

func writeFile(handle *tar.Writer, rootPath, relPath string) error {
	var file *os.File
	var stat os.FileInfo
	var buffer []byte
	var err error
	if file, err = os.OpenFile(rootPath, os.O_RDONLY, os.ModePerm); err != nil {
		return err
	}
	if stat, err = file.Stat(); err != nil {
		return err
	}
	file.Close()
	header := &tar.Header{
		Name: relPath,
		Size: stat.Size(),
	}
	if err := handle.WriteHeader(header); err != nil {
		return err
	}
	if buffer, err = ioutil.ReadFile(rootPath); err != nil {
		return err
	}
	if _, err := handle.Write(buffer); err != nil {
		return err
	}
	return nil
}