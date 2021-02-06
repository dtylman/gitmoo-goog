package downloader

import (
	"encoding/json"

	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
)

//LibraryItem Google Photo item and meta data
type LibraryItem struct {
	//Google Photos item
	photoslibrary.MediaItem
	//Actual file name that was used, without a path
	UsedFileName string
}

//MarshalJSON marshal as json
func (l *LibraryItem) MarshalJSON() ([]byte, error) {
	data, err := l.MediaItem.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	m["UsedFileName"] = l.UsedFileName
	return json.Marshal(m)
}
