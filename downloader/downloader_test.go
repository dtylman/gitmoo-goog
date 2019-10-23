package downloader_test

import (
	"encoding/json"
	"testing"

	photoslibrary "google.golang.org/api/photoslibrary/v1"
)

// TestMediaItemFileName Test to ensure that JSON with `FileName` will be
// populated under Media Item File Name
func TestMediaItemFileName(t *testing.T) {
	data := `
	{
		"baseUrl": "https://lh3.googleusercontent.com/1234",
		"id": "1234",
		"mediaMetadata": {
			"creationTime": "2019-10-13T17:33:43Z",
			"height": "3024",
			"photo": {
				"apertureFNumber": 1.7,
				"cameraMake": "motorola",
				"cameraModel": "Moto G (5) Plus",
				"focalLength": 4.28,
				"isoEquivalent": 400
			},
			"width": "4032"
		},
		"mimeType": "image/jpeg",
		"productUrl": "https://photos.google.com/1234",
		"filename": "IMG_1234.jpg"
	}
	`
	item := new(photoslibrary.MediaItem)
	json.Unmarshal([]byte(data), item)

	if item.FileName != "IMG_1234.jpg" {
		t.Errorf("photoslibrary.MediaItem.FileName = %v; want \"IMG_1234.jpg\"", item.FileName)
	}
}