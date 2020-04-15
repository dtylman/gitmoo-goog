package deb

import (
	"bytes"
	"fmt"
	"reflect"
)

//Control represents debian control structure
type Control struct {
	Package      string `json:"-"`
	Version      string `json:"-"`
	Source       string `json:"source"`
	Depends      string `json:"depends"`
	Architecture string `json:"-"`
	Maintainer   string `json:"maintainer"`
	Conflicts    string `json:"conflicts"`
	Section      string `json:"section"`
	Homepage     string `json:"homepage"`
	Description  string `json:"description"`
}

//Bytes marshal control structure as bytes
func (c *Control) bytes() []byte {
	buff := new(bytes.Buffer)
	val := reflect.ValueOf(c).Elem()
	for i := 0; i < val.NumField(); i++ {
		value := val.Field(i).String()
		if value != "" {
			buff.WriteString(fmt.Sprintf("%s: %v\n", val.Type().Field(i).Name, value))
		}
	}
	return buff.Bytes()
}
