package downloader

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	photoslibrary "google.golang.org/api/photoslibrary/v1"
)

//Options defines downloader options
var Options struct {
	//BackupFolderis the backup folder
	BackupFolder string
	//MaxItems how many items to download
	MaxItems int
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

func getFileNameByTime(item *photoslibrary.MediaItem) (string, error) {
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err != nil {
		log.Println(err)
		return "", err
	}
	year := strconv.Itoa(t.Year())
	month := t.Month().String()
	name := fmt.Sprintf("%v_%v", t.Day(), item.Id[len(item.Id)-8:])
	return filepath.Join(Options.BackupFolder, year, month, name), nil
}
func getFileNameByHash(item *photoslibrary.MediaItem) string {
	hasher := md5.New()
	hasher.Write([]byte(item.Id))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return filepath.Join(Options.BackupFolder, hash[:4], hash[4:8], hash[8:])
}

func getFileName(item *photoslibrary.MediaItem) string {
	fileName, err := getFileNameByTime(item)
	if err != nil {
		fileName = getFileNameByHash(item)
	}
	return fileName
}

func createJSON(item *photoslibrary.MediaItem, fileName string) error {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		log.Printf("Creating '%v' ", fileName)
		bytes, err := item.MarshalJSON()
		if err != nil {
			return err
		}
		err = os.MkdirAll(filepath.Dir(fileName), 0700)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(fileName, bytes, 0644)
	}
	return nil
}

func createImage(item *photoslibrary.MediaItem, fileName string) error {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		url := fmt.Sprintf("%v=w%v-h%v", item.BaseUrl, item.MediaMetadata.Width, item.MediaMetadata.Height)
		output, err := os.Create(fileName)
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

		log.Printf("Downloaded '%v' (%v)", fileName, humanize.Bytes(uint64(n)))
		stats.downloaded++
		stats.totalsize += uint64(n)
		return nil
	}
	return nil
}

func downloadItem(svc *photoslibrary.Service, item *photoslibrary.MediaItem) error {
	name := getFileName(item)
	imageName := name
	jsonName := name + ".json"
	ext, _ := mime.ExtensionsByType(item.MimeType)
	if len(ext) > 0 {
		imageName += ext[0]
	}
	err := createJSON(item, jsonName)
	if err != nil {
		return err
	}

	return createImage(item, imageName)
}

//DownloadAll downloads all files
func DownloadAll(svc *photoslibrary.Service) error {
	hasMore := true
	stats.downloaded = 0
	stats.errors = 0
	stats.total = 0
	stats.totalsize = 0
	req := &photoslibrary.SearchMediaItemsRequest{PageSize: int64(Options.MaxItems), AlbumId: Options.AlbumID}
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
