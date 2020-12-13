// Package downloader Options defines downloader options
package downloader

// Options Defines downloader various options
type Options struct {
	//BackupFolderis the backup folder
	BackupFolder string
	//FolderFormat time format used to format folder structure
	FolderFormat string
	//UseFileName use file name when uploaded to Google Photos
	UseFileName bool
	//OriginalFiles retain EXIF metadata on downloaded images. Location information is not included.
	IncludeEXIF bool
	//MaxItems how many items to download
	MaxItems int
	//number of items to download on per API call
	PageSize int
	//Throttle is time to wait between API calls
	Throttle int
	//DownloadThrottle is the rate to limit downloading of items (KB/sec)
	DownloadThrottle float64
	//ConcurrentDownloads is the number of downloads that can happen at once
	ConcurrentDownloads int
	//Google photos AlbumID
	AlbumID string
	//CredentialsFile Google API credentials.json file
	CredentialsFile string
	//TokenFile Google oauth client token.json file
	TokenFile string
}
