package files

import (
	"os"
	"path/filepath"
)

//Container ...
type Container struct {
	Files []string
}

//New ...
func New(path string) (*Container, error) {
	c := new(Container)
	err := c.walk(path)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Container) walk(path string) error {
	return filepath.Walk(path, c.add)
}

func (c *Container) add(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.Mode().IsRegular() {
		c.Files = append(c.Files, path)
		return nil
	}
	return nil
}
