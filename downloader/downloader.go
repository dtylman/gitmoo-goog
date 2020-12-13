package downloader

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fujiwara/shapeio"
	errgroup "golang.org/x/sync/errgroup"
	gensupport "google.golang.org/api/gensupport"
	photoslibrary "google.golang.org/api/photoslibrary/v1"
)

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

// Downloader Struct for downloading photos into managed folders, use factory
// method `NewDownloader` to create
type Downloader struct {
	waitGroup                  *errgroup.Group
	concurrentDownloadRoutines chan struct{}
	stats                      *Stats
	Options                    *Options
}

// NewDownloader factory to create a Downloader instance with defaults
func NewDownloader() *Downloader {
	downloader := new(Downloader)
	downloader.waitGroup = new(errgroup.Group)
	downloader.stats = new(Stats)

	downloader.Options = new(Options)
	downloader.Options.BackupFolder, _ = os.Getwd()
	downloader.Options.FolderFormat = filepath.Join("2006", "January")
	downloader.Options.ConcurrentDownloads = 1

	return downloader
}

// getFolderPath Path of the to store JSON and image files for the particular MediaItem
func (d *Downloader) getFolderPath(item *photoslibrary.MediaItem) string {
	//TODO Check that item.MediaMetadata exists
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err != nil {
		//Default to an epoch if cannot parse time
		t, err = time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")
	}

	return filepath.Join(d.Options.BackupFolder, t.Format(d.Options.FolderFormat))
}

// createFileName Get the full path to the image file including what conflict position we are at
func (d *Downloader) createFileName(item *LibraryItem, conflict int) string {
	fileName := d.getImageFilePath(item)
	if conflict > 0 {
		fileExtension := filepath.Ext(fileName)
		index := strings.LastIndex(fileName, fileExtension)
		fileName = fileName[0:index] + " (" + fmt.Sprintf("%d", conflict) + ")" + fileExtension
	}

	return filepath.Base(fileName)
}

// isConflictingFilePath Check if the image file already exists
func (d *Downloader) isConflictingFilePath(item *LibraryItem) bool {
	_, err := os.Stat(d.getImageFilePath(item))

	return err == nil
}

// getLegacyPrefixFilePathByTime Build a file path based on the image creation
// time, file extension will need to be appened after
func (d *Downloader) getLegacyPrefixFilePathByTime(item *photoslibrary.MediaItem) (string, error) {
	//TODO check that item.MediaMetadata and item.Id exist
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err != nil {
		return "", err
	}
	//TODO Assuming item.Id is over a certain length without checking
	name := fmt.Sprintf("%v_%v", t.Day(), item.Id[len(item.Id)-8:])
	return filepath.Join(d.getFolderPath(item), name), nil
}

// getLegacyPrefixFilePathByHash Build a file path when missing a image
// creation time based on a MD5 hash of the Media Item ID, file extension will
// need to be appened after
func (d *Downloader) getLegacyPrefixFilePathByHash(item *photoslibrary.MediaItem) string {
	hasher := md5.New()
	hasher.Write([]byte(item.Id))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return filepath.Join(d.Options.BackupFolder, hash[:4], hash[4:8], hash[8:])
}

// getLegacyPrefixFilePath Build a file path based on legacy naming convention
// of using Media Item ID, file extension will need to be appened after
func (d *Downloader) getLegacyPrefixFilePath(item *photoslibrary.MediaItem) string {
	//Legacy file names
	fileName, err := d.getLegacyPrefixFilePathByTime(item)
	if err != nil {
		//Must return since this provides its own folder paths
		fileName = d.getLegacyPrefixFilePathByHash(item)
	}

	return fileName
}

// getImageFilePath Get the file path for the image
func (d *Downloader) getImageFilePath(item *LibraryItem) string {
	var fileName string

	if d.Options.UseFileName {
		fileName = item.UsedFileName
		if fileName == "" {
			fileName = item.FileName
		}

		fileName = filepath.Join(d.getFolderPath(&item.MediaItem), fileName)
	} else {
		fileName = d.getLegacyPrefixFilePath(&item.MediaItem)

		//Append the file extension based on the mime type
		ext, _ := mime.ExtensionsByType(item.MimeType)
		if len(ext) > 0 {
			fileName += ext[0]
		}
	}

	return fileName
}

// getJSONFilePath Get the full path to the JSON file representing the MediaItem
func (d *Downloader) getJSONFilePath(item *photoslibrary.MediaItem) string {
	if d.Options.UseFileName {
		//TODO item.Id could be missing
		return filepath.Join(d.getFolderPath(item), "."+item.Id+".json")
	}

	return d.getLegacyPrefixFilePath(item) + ".json"
}

// loadJSON Load the JSON file into LibraryItem
func (d *Downloader) loadJSON(filePath string) (*LibraryItem, error) {
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
func (d *Downloader) createJSON(item *LibraryItem, filePath string) error {
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

// downloadImage TODO
func (d *Downloader) downloadImage(item *LibraryItem, filePath string) error {
	var url string

	if strings.HasPrefix(strings.ToLower(item.MediaItem.MimeType), "video") {
		url = item.MediaItem.BaseUrl + "=dv"
	} else {
		if d.Options.IncludeEXIF {
			url = fmt.Sprintf("%v=d", item.MediaItem.BaseUrl)
		} else {
			url = fmt.Sprintf("%v=w%v-h%v", item.MediaItem.BaseUrl, item.MediaItem.MediaMetadata.Width, item.MediaItem.MediaMetadata.Height)
		}
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

	//Limit download rate
	rateLimitedReader := shapeio.NewReader(response.Body)
	if d.Options.DownloadThrottle > 0.0 {
		rateLimitedReader.SetRateLimit((d.Options.DownloadThrottle * 1024) / float64(d.Options.ConcurrentDownloads))
	}

	n, err := io.Copy(output, rateLimitedReader)
	if err != nil {
		return err
	}

	// close file to prevent conflicts with writing new timestamp in next step
	output.Close()

	//If timestamp is available, set access time to current timestamp and set modified time to the time the item was first created (not when it was uploaded to Google Photos)
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err == nil {
		err = os.Chtimes(filePath, time.Now(), t)
		if err != nil {
			return errors.New("failed writing timestamp to file: " + err.Error())
		}
	}

	log.Printf("Downloaded '%v' [saved as '%v'] (%v)", item.FileName, item.UsedFileName, humanize.Bytes(uint64(n)))

	d.stats.UpdateStatsDownloaded(uint64(n), 1)

	//Inform channel download is complete
	<-d.concurrentDownloadRoutines

	return nil
}

// createImage Download the image file if it does not already exist
func (d *Downloader) createImage(item *LibraryItem, filePath string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		//Touch file before downloading (to avoid file name conflicts)
		err := ioutil.WriteFile(filePath, []byte{}, 0644)
		if err != nil {
			return err
		}

		//Wait till room on channel to start download
		d.concurrentDownloadRoutines <- struct{}{}
		d.waitGroup.Go(func() error {
			return d.downloadImage(item, filePath)
		})
	} else {
		log.Printf("Skipping '%v' [saved as '%v']", item.FileName, item.UsedFileName)
	}
	return nil
}

// downloadItem TODO
func (d *Downloader) downloadItem(svc *photoslibrary.Service, item *photoslibrary.MediaItem) error {
	jsonFilePath := d.getJSONFilePath(item)

	libraryItem, err := d.loadJSON(jsonFilePath)
	if err != nil {
		return err
	}
	if libraryItem == nil {
		libraryItem = new(LibraryItem)
		libraryItem.MediaItem = *item

		//Create non-conflicting file name
		for conflict := 0; true; conflict++ {
			libraryItem.UsedFileName = d.createFileName(libraryItem, conflict)
			if !d.isConflictingFilePath(libraryItem) {
				break
			}
		}
	}

	err = d.createJSON(libraryItem, jsonFilePath)
	if err != nil {
		return err
	}
	return d.createImage(libraryItem, d.getImageFilePath(libraryItem))
}

// DownloadAll downloads all files
func (d *Downloader) DownloadAll(svc *photoslibrary.Service) error {
	hasMore := true
	sleepTime := time.Duration(time.Second * time.Duration(d.Options.Throttle))

	//Setup channel buffer to limit downloads
	d.concurrentDownloadRoutines = make(chan struct{}, d.Options.ConcurrentDownloads)

	req := &photoslibrary.SearchMediaItemsRequest{PageSize: int64(d.Options.PageSize), AlbumId: d.Options.AlbumID}
	for hasMore {
		items, err := svc.MediaItems.Search(req).Do()
		if err != nil {
			return err
		}
		for _, m := range items.MediaItems {
			d.stats.UpdateStatsTotal(1)
			err = d.downloadItem(svc, m)
			if err != nil {
				log.Printf("Failed to download '%v' [id %v]: %v", m.FileName, m.Id, err)
				d.stats.UpdateStatsError(1)
			}

			if d.stats.Total >= d.Options.MaxItems {
				hasMore = false
				break
			}
		}
		req.PageToken = items.NextPageToken
		if req.PageToken == "" {
			hasMore = false
		}

		//Wait for all downloads in group to complete, return if any errors
		err = d.waitGroup.Wait()
		if err != nil {
			return err
		}

		if hasMore {
			log.Printf("Processed: %v, Downloaded: %v, Errors: %v, Total Size: %v", d.stats.Total, d.stats.Downloaded, d.stats.Errors, humanize.Bytes(d.stats.TotalSize))
			time.Sleep(sleepTime)
		}
	}

	log.Printf("Finished: %v, Downloaded: %v, Errors: %v, Total Size: %v", d.stats.Total, d.stats.Downloaded, d.stats.Errors, humanize.Bytes(d.stats.TotalSize))
	return nil
}
