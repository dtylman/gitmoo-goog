package deb

import (
	"errors"
	"time"

	"fmt"
	"os"
	"path/filepath"

	"strings"

	"github.com/blakesmith/ar"
	"github.com/dtylman/gopack/files"
)

const (
	//AMD64 amd664 arch
	AMD64      = "amd64"
	debControl = "control.tar.gz"
	debData    = "data.tar.gz"
	debBinary  = "debian-binary"
)

//Deb represents a deb package
type Deb struct {
	Data     *canonical `json:"-"`
	Control  *canonical `json:"-"`
	Info     Control    `json:"control"`
	PreInst  string     `json':"pre_inst"`
	PostInst string     `json':"post_inst"`
	PreRm    string     `json:"pre_rm"`
	PostRm   string     `json:"post_rm"`
}

//New creates new deb writer
func New(name, version, revision, arch string) (*Deb, error) {
	deb := new(Deb)
	deb.Info.Package = name
	deb.Info.Version = version
	if revision != "" {
		deb.Info.Version += "-" + revision
	}
	deb.Info.Architecture = arch
	var err error
	deb.Data, err = newCanonical()
	if err != nil {
		return nil, err
	}
	deb.Control, err = newCanonical()
	if err != nil {
		return nil, err
	}
	return deb, nil
}

//Create creates the deb file
func (d *Deb) Create(folder string) (string, error) {
	if d.Info.Package == "" {
		return "", errors.New("Package name cannot be empty")
	}
	err := d.Control.AddEmptyFolder("./")
	if err != nil {
		return "", err
	}
	err = d.Control.AddBytes(d.Info.bytes(), "./control")
	if err != nil {
		return "", err
	}
	err = d.Control.AddBytes(d.Data.md5s.Bytes(), "./md5sums")
	if err != nil {
		return "", err
	}
	if d.PostInst != "" {
		d.Control.AddBytes([]byte(d.PostInst), "postinst")
	}
	if d.PreInst != "" {
		d.Control.AddBytes([]byte(d.PreInst), "preinst")

	}
	if d.PostRm != "" {
		d.Control.AddBytes([]byte(d.PostRm), "postrm")

	}
	if d.PreRm != "" {
		d.Control.AddBytes([]byte(d.PreRm), "prerm")
	}
	fileName := filepath.Join(folder, fmt.Sprintf("%s_%s_%s.deb", d.Info.Package, d.Info.Version, d.Info.Architecture))
	debFile, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer debFile.Close()
	ar := ar.NewWriter(debFile)
	err = ar.WriteGlobalHeader()
	if err != nil {
		return "", err
	}

	err = d.addBinary(ar)
	if err != nil {
		return "", err
	}
	err = d.Control.write(ar, debControl)
	if err != nil {
		return "", err
	}
	err = d.Data.write(ar, debData)
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func (d *Deb) addBinary(writer *ar.Writer) error {
	body := []byte("2.0\n")
	header := new(ar.Header)
	header.Name = debBinary
	header.Mode = 0664
	header.Size = int64(len(body))
	header.ModTime = time.Now()
	err := writer.WriteHeader(header)
	if err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}

//AddFile adds a file to package
func (d *Deb) AddFile(sourcePath string, targetPath string) error {
	return d.Data.AddFile(sourcePath, targetPath)
}

//AddEmptyFolder adds empty folder to package
func (d *Deb) AddEmptyFolder(name string) error {
	return d.Data.AddEmptyFolder(name)
}

//AddFolder adds folder to package
func (d *Deb) AddFolder(path string, prefix string) error {
	fc, err := files.New(path)
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(path)
	for _, path := range fc.Files {
		targetPath := filepath.Join(prefix, strings.TrimPrefix(path, baseDir))
		err = d.AddFile(path, targetPath)
		if err != nil {
			return err
		}
	}
	return nil
}
