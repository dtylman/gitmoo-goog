package downloader

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
)

// tempPath Create a temporary path (must be removed after)
func tempPath() string {
	path, _ := ioutil.TempDir("", "gitmoo-goog-test")
	return path
}

func TestMarshallJSON(t *testing.T) {
	item := new(LibraryItem)
	item.UsedFileName = "test.jpg"

	bytes, err := item.MarshalJSON()
	if err != nil {
		t.Fatalf("%v", err)
	}
	have := string(bytes)
	want := "{\"UsedFileName\":\"test.jpg\"}"

	if have != want {
		t.Errorf("LibraryItem.MarshallJSON = %v; want %v", have, want)
	}
}

func TestGetFolderPath(t *testing.T) {
	t.Run("Missing", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.BackupFolder = tempPath()
		defer os.RemoveAll(downloader.Options.BackupFolder)

		item := new(photoslibrary.MediaItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)

		have := downloader.getFolderPath(item)
		want := filepath.Join(downloader.Options.BackupFolder, "1970", "January")
		if have != want {
			t.Errorf("downloader.getFolderPath() = %v; want %v", have, want)
		}
	})

	t.Run("Date", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.BackupFolder = tempPath()
		defer os.RemoveAll(downloader.Options.BackupFolder)

		item := new(photoslibrary.MediaItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := downloader.getFolderPath(item)
		want := filepath.Join(downloader.Options.BackupFolder, "2019", "October")
		if have != want {
			t.Errorf("downloader.getFolderPath() = %v; want %v", have, want)
		}
	})

	t.Run("Format", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.BackupFolder = tempPath()
		defer os.RemoveAll(downloader.Options.BackupFolder)
		downloader.Options.FolderFormat = filepath.Join("2006", "01")

		item := new(photoslibrary.MediaItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := downloader.getFolderPath(item)
		want := filepath.Join(downloader.Options.BackupFolder, "2019", "10")
		if have != want {
			t.Errorf("downloader.getFolderPath() = %v; want %v", have, want)
		}
	})
}

func TestCreateFileName(t *testing.T) {
	t.Run("Legacy Hash", func(t *testing.T) {
		downloader := NewDownloader()

		item := new(LibraryItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.Filename = "test.jpg"

		have := downloader.createFileName(item, 0)
		want := "8f00b204e9800998ecf8427e"

		if have != want {
			t.Errorf("downloader.createFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Legacy Time", func(t *testing.T) {
		downloader := NewDownloader()

		item := new(LibraryItem)
		item.Id = "12345678901234567890"
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := downloader.createFileName(item, 0)
		want := "13_34567890"

		if have != want {
			t.Errorf("downloader.createFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Legacy Time With Mime Type", func(t *testing.T) {
		downloader := NewDownloader()

		item := new(LibraryItem)
		item.Id = "12345678901234567890"
		item.MimeType = "image/jpeg"
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := downloader.createFileName(item, 0)
		want := "13_34567890.jpe"

		if have != want {
			t.Errorf("downloader.createFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Use File Name", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.UseFileName = true

		item := new(LibraryItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.Filename = "test.jpg"

		have := downloader.createFileName(item, 0)
		want := item.Filename

		if have != want {
			t.Errorf("downloader.createFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Use File Name Uses UsedFileName", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.UseFileName = true

		item := new(LibraryItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.Filename = "test.jpg"
		item.UsedFileName = "test (1).jpg"

		have := downloader.createFileName(item, 0)
		want := item.UsedFileName

		if have != want {
			t.Errorf("downloader.createFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Conflict", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.UseFileName = true

		item := new(LibraryItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.Filename = "test.jpg"

		have := downloader.createFileName(item, 1)
		want := "test (1).jpg"

		if have != want {
			t.Errorf("downloader.createFileName() = %v; want %v", have, want)
		}
	})
}

func TestIsConflictingFilePath(t *testing.T) {
	t.Run("Conflicting", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.UseFileName = true
		downloader.Options.BackupFolder = tempPath()
		defer os.RemoveAll(downloader.Options.BackupFolder)

		item := new(LibraryItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.Filename = "test.jpg"

		//Create directory and touch the file
		path := downloader.getImageFilePath(item)
		err := os.MkdirAll(filepath.Dir(path), 0700)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = ioutil.WriteFile(path, []byte{}, 0644)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !downloader.isConflictingFilePath(item) {
			t.Errorf("downloader.isConflictingFilePath() = %v; want %v", false, true)
		}
	})

	t.Run("Not Conflicting", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.UseFileName = true
		downloader.Options.BackupFolder = tempPath()
		defer os.RemoveAll(downloader.Options.BackupFolder)

		item := new(LibraryItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.Filename = "test.jpg"

		if downloader.isConflictingFilePath(item) {
			t.Errorf("downloader.isConflictingFilePath() = %v; want %v", true, false)
		}
	})
}

func TestGetJSONFilePath(t *testing.T) {
	t.Run("Legacy Hash", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.BackupFolder = tempPath()
		defer os.RemoveAll(downloader.Options.BackupFolder)

		item := new(photoslibrary.MediaItem)
		item.Id = "12345678901234567890"
		item.MediaMetadata = new(photoslibrary.MediaMetadata)

		have := downloader.getJSONFilePath(item)
		want := filepath.Join(downloader.Options.BackupFolder, "fd85", "e62d", "9beb45428771ec688418b271.json")

		if have != want {
			t.Errorf("downloader.getJSONFilePath() = %v; want %v", have, want)
		}
	})

	t.Run("Legacy Time", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.BackupFolder = tempPath()
		defer os.RemoveAll(downloader.Options.BackupFolder)

		item := new(photoslibrary.MediaItem)
		item.Id = "12345678901234567890"
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := downloader.getJSONFilePath(item)
		want := filepath.Join(downloader.Options.BackupFolder, "2019", "October", "13_34567890.json")

		if have != want {
			t.Errorf("downloader.getJSONFilePath() = %v; want %v", have, want)
		}
	})

	t.Run("Use File Name", func(t *testing.T) {
		downloader := NewDownloader()
		downloader.Options.UseFileName = true
		downloader.Options.BackupFolder = tempPath()
		defer os.RemoveAll(downloader.Options.BackupFolder)

		item := new(photoslibrary.MediaItem)
		item.Id = "12345678901234567890"
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := downloader.getJSONFilePath(item)
		want := filepath.Join(downloader.Options.BackupFolder, "2019", "October", ".12345678901234567890.json")

		if have != want {
			t.Errorf("downloader.getJSONFilePath() = %v; want %v", have, want)
		}
	})
}

func TestCreateAndLoadJSON(t *testing.T) {
	downloader := NewDownloader()
	downloader.Options.BackupFolder = tempPath()
	defer os.RemoveAll(downloader.Options.BackupFolder)
	downloader.Options.UseFileName = true

	item := new(LibraryItem)
	item.Id = "12345678901234567890"
	item.MediaMetadata = new(photoslibrary.MediaMetadata)

	filePath := filepath.Join(downloader.Options.BackupFolder, "test.json")

	err := downloader.createJSON(item, filePath)
	if err != nil {
		t.Fatalf("%v", err)
	}

	newItem, err := downloader.loadJSON(filePath)
	if err != nil {
		t.Fatalf("%v", newItem)
	}

	if item.Id != newItem.Id {
		t.Errorf("downloader.createJSON() = %v; want %v", newItem, item)
	}
}

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

	if item.Filename != "IMG_1234.jpg" {
		t.Errorf("photoslibrary.MediaItem.FileName = %v; want \"IMG_1234.jpg\"", item.Filename)
	}
}
