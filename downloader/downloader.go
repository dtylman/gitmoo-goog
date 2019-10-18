package downloader

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"encoding/json"
	"crypto/md5"
	"encoding/hex"
	"mime"

	"github.com/dustin/go-humanize"
	photoslibrary "google.golang.org/api/photoslibrary/v1"
	gensupport "google.golang.org/api/gensupport"
)

//Options defines downloader options
var Options struct {
	//BackupFolderis the backup folder
	BackupFolder string
	//FolderFormat time format used to format folder structure
	FolderFormat string
	//UseFileName use file name when uploaded to Google Photos
	UseFileName bool
	//MaxItems how many items to download
	MaxItems int
	//number of items to download on per API call
	PageSize int
	//Throtthle is time to wait between API calls
	Throttle int
	//Google photos AlbumID
	AlbumID string
}

var stats struct {
	total      int
	errors     int
	totalsize  uint64
	downloaded int
}

//LibraryItem Google Photo item and meta data
type LibraryItem struct {
	//Google Photos item
	photoslibrary.MediaItem

	//Actual file name that was used, without a path
	UsedFileName string
}

//MarshalJSON Convert LibraryItem to JSON utlizing pre-built Google marshalling
func (s *LibraryItem) MarshalJSON() ([]byte, error) {
	type NoMethod LibraryItem
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, []string{}, []string{})
}

func init() {
	Options.BackupFolder, _ = os.Getwd()
	Options.FolderFormat = filepath.Join("2006", "January")
}

// getFolderPath Path of the to store JSON and image files for the particular MediaItem
func getFolderPath(item *photoslibrary.MediaItem) string {
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err != nil {
		//Default to an epoch if cannot parse time
		t, err = time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")
	}

	return filepath.Join(Options.BackupFolder, t.Format(Options.FolderFormat))
}

// createFileName Get the full path to the image file including what conflict position we are at
func createFileName(item *LibraryItem, conflict int) string {
	fileName := getImageFilePath(item)
	if conflict > 0 {
		fileExtension := filepath.Ext(fileName)
		index := strings.LastIndex(fileName, fileExtension)
		fileName = fileName[0:index] + " (" + fmt.Sprintf("%d", conflict) + ")" + fileExtension
	}

	return filepath.Base(fileName);
}

// isConflictingFilePath Check if the image file already exists
func isConflictingFilePath(item *LibraryItem) bool {
	_, err := os.Stat(getImageFilePath(item))

	return err == nil
}

// getLegacyPrefixFilePathByTime Build a file path based on the image creation
// time, file extension will need to be appened after
func getLegacyPrefixFilePathByTime(item *photoslibrary.MediaItem) (string, error) {
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%v_%v", t.Day(), item.Id[len(item.Id)-8:])
	return filepath.Join(getFolderPath(item), name), nil
}

// getLegacyPrefixFilePathByHash Build a file path when missing a image
// creation time based on a MD5 hash of the Media Item ID, file extension will
// need to be appened after
func getLegacyPrefixFilePathByHash(item *photoslibrary.MediaItem) string {
	hasher := md5.New()
	hasher.Write([]byte(item.Id))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return filepath.Join(Options.BackupFolder, hash[:4], hash[4:8], hash[8:])
}

// getLegacyPrefixFilePath Build a file path based on legacy naming convention
// of using Media Item ID, file extension will need to be appened after
func getLegacyPrefixFilePath(item *photoslibrary.MediaItem) string {
	//Legacy file names
	fileName, err := getLegacyPrefixFilePathByTime(item)
	if err != nil {
		//Must return since this provides its own folder paths
		fileName = getLegacyPrefixFilePathByHash(item)
	}

	return fileName
}

// getImageFilePath Get the file path for the image
func getImageFilePath(item *LibraryItem) string {
	var fileName string

	if Options.UseFileName {
		fileName = item.UsedFileName
		if fileName == "" {
			fileName = item.FileName
		}

		fileName = filepath.Join(getFolderPath(&item.MediaItem), fileName);
	} else {
		fileName = getLegacyPrefixFilePath(&item.MediaItem)

		//Append the file extension based on the mime type
		ext, _ := mime.ExtensionsByType(item.MimeType)
		if len(ext) > 0 {
			fileName += ext[0]
		}
	}

	return fileName
}

// getJSONFilePath Get the full path to the JSON file representing the MediaItem
func getJSONFilePath(item *photoslibrary.MediaItem) string {
	if Options.UseFileName {
		return filepath.Join(getFolderPath(item), "." + item.Id + ".json");
	} else {
		return getLegacyPrefixFilePath(item) + ".json"
	}
}

// loadJSON Load the JSON file into LibraryItem
func loadJSON(filePath string) (*LibraryItem, error) {
	info, err := os.Stat(filePath)

	if err == nil && info != nil {
		bytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		item := new(LibraryItem)
		err = json.Unmarshal(bytes, item)
		if err != nil {
			return nil, err
		}
		return item, nil
	}

	return nil, nil
}

// createJSON create a JSON file if it does not already exist
func createJSON(item *LibraryItem, filePath string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		log.Printf("Creating JSON for '%v' ", item.UsedFileName)
		bytes, err := item.MarshalJSON()
		if err != nil {
			return err
		}
		err = os.MkdirAll(filepath.Dir(filePath), 0700)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(filePath, bytes, 0644)
	}
	return nil
}

// createImage Download the image file if it does not already exist
func createImage(item *LibraryItem, filePath string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		var url string
		if strings.HasPrefix(strings.ToLower(item.MediaItem.MimeType), "video") {
			url = item.MediaItem.BaseUrl + "=dv"
		} else {
			url = fmt.Sprintf("%v=w%v-h%v", item.MediaItem.BaseUrl, item.MediaItem.MediaMetadata.Width, item.MediaItem.MediaMetadata.Height)
		}
		output, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer output.Close()

		response, err := http.Get(url)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		n, err := io.Copy(output, response.Body)
		if err != nil {
			return err
		}

		log.Printf("Downloaded '%v' [saved as '%v'] (%v)", item.FileName, item.UsedFileName, humanize.Bytes(uint64(n)))
		stats.downloaded++
		stats.totalsize += uint64(n)
	} else {
		log.Printf("Skipping '%v' [saved as '%v']", item.FileName, item.UsedFileName)
	}
	return nil
}

func downloadItem(svc *photoslibrary.Service, item *photoslibrary.MediaItem) error {
	jsonFilePath := getJSONFilePath(item)

	libraryItem, err := loadJSON(jsonFilePath)
	if err != nil {
		return err
	}
	if libraryItem == nil {
		libraryItem = new(LibraryItem)
		libraryItem.MediaItem = *item
		libraryItem.UsedFileName = createFileName(libraryItem, 0)

		for conflict := 0; isConflictingFilePath(libraryItem); conflict++ {
			libraryItem.UsedFileName = createFileName(libraryItem, conflict)
		}
	}

	err = createJSON(libraryItem, jsonFilePath)
	if err != nil {
		return err
	}
	return createImage(libraryItem, getImageFilePath(libraryItem))
}

//DownloadAll downloads all files
func DownloadAll(svc *photoslibrary.Service) error {
	hasMore := true
	stats.downloaded = 0
	stats.errors = 0
	stats.total = 0
	stats.totalsize = 0
	req := &photoslibrary.SearchMediaItemsRequest{PageSize: int64(Options.PageSize), AlbumId: Options.AlbumID}
	for hasMore {
		sleepTime := time.Duration(time.Second * time.Duration(Options.Throttle))
		log.Printf("Processed: %v, Downloaded: %v, Errors: %v, Total Size: %v, Waiting %v", stats.total, stats.downloaded, stats.errors, humanize.Bytes(stats.totalsize), sleepTime)
		time.Sleep(sleepTime)
		items, err := svc.MediaItems.Search(req).Do()
		if err != nil {
			return err
		}
		for _, m := range items.MediaItems {
			stats.total++
			if stats.total > Options.MaxItems {
				hasMore = false
				break
			}
			err = downloadItem(svc, m)
			if err != nil {
				log.Printf("Failed to download '%v' [id %v]: %v", m.FileName, m.Id, err)
				stats.errors++
			}
		}
		req.PageToken = items.NextPageToken
		if req.PageToken == "" {
			hasMore = false
		}
	}

	log.Printf("Processed: %v, Downloaded: %v, Errors: %v, Total Size: %v",
		stats.total, stats.downloaded, stats.errors, humanize.Bytes(stats.totalsize))
	return nil
}
