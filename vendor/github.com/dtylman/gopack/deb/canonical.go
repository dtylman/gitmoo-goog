package deb

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"bytes"

	"path/filepath"
	"strings"

	"github.com/blakesmith/ar"
)

type canonical struct {
	file         *os.File
	zip          *gzip.Writer
	tarWriter    *tar.Writer
	md5s         bytes.Buffer
	emptyFolders map[string]bool
}

func newCanonical() (*canonical, error) {
	c := new(canonical)
	var err error
	c.file, err = ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	c.zip = gzip.NewWriter(c.file)
	c.tarWriter = tar.NewWriter(c.zip)
	c.emptyFolders = make(map[string]bool)
	return c, nil
}

func (c *canonical) AddBytes(data []byte, tarName string) error {
	header := new(tar.Header)
	header.Name = tarName
	header.Mode = 0664
	header.Size = int64(len(data))
	header.ModTime = time.Now()
	header.Typeflag = tar.TypeReg
	err := c.tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}
	md5 := md5.New()
	_, err = c.tarWriter.Write(data)
	if err != nil {
		return err
	}
	err = c.tarWriter.Flush()
	if err != nil {
		return err
	}
	_, err = c.md5s.WriteString(fmt.Sprintf("%x  %s\n", md5.Sum(data), header.Name))
	if err != nil {
		return err
	}
	return nil
}

func (c *canonical) AddLink(name string, linkName string) error {
	header := new(tar.Header)
	header.Name = name
	header.Linkname = linkName
	header.Mode = 0664
	header.ModTime = time.Now()
	header.Typeflag = tar.TypeSymlink
	return c.tarWriter.WriteHeader(header)
}

func (c *canonical) AddEmptyFolder(name string) error {
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return fmt.Errorf("Cannot add empty name for empty folder")
	}
	header := new(tar.Header)
	header.Name = name
	header.Mode = 0775
	header.ModTime = time.Now()
	header.Typeflag = tar.TypeDir
	return c.tarWriter.WriteHeader(header)
}

func (c *canonical) AddFile(name string, tarName string) error {
	fileInfo, err := os.Stat(name)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("%s is a directory, use AddFolder instead", name)
	}
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", name)
	}
	tarName = strings.TrimPrefix(tarName, "/")
	err = c.ensureFilePath(tarName)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(fileInfo, "")
	if tarName != "" {
		header.Name = tarName
	}
	if err != nil {
		return err
	}
	err = c.tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	md5 := md5.New()
	reader := io.TeeReader(file, md5)
	_, err = io.Copy(c.tarWriter, reader)
	if err != nil {
		return err
	}
	err = c.tarWriter.Flush()
	if err != nil {
		return err
	}
	err = c.zip.Flush()
	if err != nil {
		return err
	}
	_, err = c.md5s.WriteString(fmt.Sprintf("%x  %s\n", md5.Sum(nil), header.Name))
	if err != nil {
		return err
	}
	return nil
}

func (c *canonical) close() error {
	err := c.tarWriter.Flush()
	if err != nil {
		return err
	}
	err = c.tarWriter.Close()
	if err != nil {
		return err
	}
	err = c.zip.Close()
	if err != nil {
		return err
	}
	err = c.file.Close()
	if err != nil {
		return err
	}
	return nil
}

func (c *canonical) ensureFilePath(path string) error {
	items := strings.Split(path, string(filepath.Separator))
	folder := "/"
	for i := 0; i < len(items)-1; i++ {
		folder = filepath.Join(folder, items[i])
		_, ok := c.emptyFolders[folder]
		if !ok {
			err := c.AddEmptyFolder(folder)
			if err != nil {
				return err
			}
			c.emptyFolders[folder] = true
		}
	}
	return nil
}

// cannot use io.copy because ar writes more data than wha'ts read
func ioCopy(writer io.Writer, reader io.Reader) error {
	buf := make([]byte, 64*1024)
	for {
		count, readErr := reader.Read(buf)
		if count > 0 {
			written, writeErr := writer.Write(buf[0:count])
			if writeErr != nil {
				return writeErr
			}
			if count > written {
				return io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (c *canonical) write(writer *ar.Writer, name string) error {
	fileName := c.file.Name()
	defer os.Remove(fileName)

	err := c.close()
	if err != nil {
		return err
	}
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return err
	}
	in, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer in.Close()
	header := new(ar.Header)
	header.Name = name
	header.Size = fileInfo.Size()
	header.Mode = 0664
	header.ModTime = time.Now()
	err = writer.WriteHeader(header)
	if err != nil {
		return err
	}

	err = ioCopy(writer, in)
	if err != nil {
		return err
	}
	return nil
}
