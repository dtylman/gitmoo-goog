package downloader

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"encoding/json"

	"github.com/dustin/go-humanize"
	photoslibrary "google.golang.org/api/photoslibrary/v1"
	gensupport "google.golang.org/api/gensupport"
)

//Options defines downloader options
var Options struct {
	//BackupFolderis the backup folder
	BackupFolder string
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
	Item *photoslibrary.MediaItem

	//Actual file name that was used, without a path
	UsedFileName string
}

//MarshalJSON Convert LibraryItem to JSON utlizing pre-built Google marshalling
func (s *LibraryItem) MarshalJSON() ([]byte, error) {
	type NoMethod LibraryItem
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, []string{}, []string{})
}

// getFolderPath Path of the to store JSON and image files for the particular MediaItem
func getFolderPath(item *photoslibrary.MediaItem) string {
	backupFolder := Options.BackupFolder
	if backupFolder == "" {
		backupFolder, _ = os.Getwd()
	}

	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	year, month := "", ""
	if err != nil {
		year = "1970"
		month = "01"
	} else {
		year = strconv.Itoa(t.Year())
		month = fmt.Sprintf("%02d", t.Month())
	}
	return filepath.Join(backupFolder, year, month)
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

// getImageFilePath Get the file path for the image
func getImageFilePath(item *LibraryItem) string {
	return filepath.Join(getFolderPath(item.Item), item.UsedFileName);
}

// getJSONFilePath Get the full path to the JSON file representing the MediaItem
func getJSONFilePath(item *photoslibrary.MediaItem) string {
	return filepath.Join(getFolderPath(item), "." + item.Id + ".json");
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
		if strings.HasPrefix(strings.ToLower(item.Item.MimeType), "video") {
			url = item.Item.BaseUrl + "=dv"
		} else {
			url = fmt.Sprintf("%v=w%v-h%v", item.Item.BaseUrl, item.Item.MediaMetadata.Width, item.Item.MediaMetadata.Height)
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

		log.Printf("Downloaded '%v' (%v)", item.UsedFileName, humanize.Bytes(uint64(n)))
		stats.downloaded++
		stats.totalsize += uint64(n)
	} else {
		log.Printf("Skipping '%v'", item.UsedFileName)
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
		libraryItem.Item = item
		libraryItem.UsedFileName = item.FileName

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
				log.Printf("Failed to download %v: %v", m.Id, err)
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
