package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dtylman/gitmoo-goog/downloader"
	"github.com/dtylman/gitmoo-goog/version"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	photoslibrary "google.golang.org/api/photoslibrary/v1"
	"gopkg.in/natefinch/lumberjack.v2"
)

var options struct {
	loop         bool
	logfile      string
	ignoreerrors bool
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokFile string) *http.Client {
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
}

func process(downloader *downloader.Downloader) error {
	b, err := ioutil.ReadFile(downloader.Options.CredentialsFile)
	if err != nil {
		log.Println("Enable photos API here: https://developers.google.com/photos/library/guides/get-started#enable-the-api")
		return fmt.Errorf("Unable to read client secret file: %v", err)
	}

	//request photos readonly access
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/photoslibrary.readonly")
	if err != nil {
		return fmt.Errorf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config, downloader.Options.TokenFile)
	log.Printf("Connecting ...")
	srv, err := photoslibrary.New(client)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Google Photos API client: %v", err)
	}
	for true {
		err := downloader.DownloadAll(srv)
		if err != nil {
			if options.ignoreerrors {
				log.Println(err)
			} else {
				return err
			}
		}
		if !options.loop {
			break
		}
	}
	return nil
}

func main() {
	workingDirectory, _ := os.Getwd()
	downloader := downloader.NewDownloader()
	log.Println("This is gitmoo-goog ver", version.Version)
	flag.BoolVar(&options.loop, "loop", false, "loops forever (use as daemon)")
	flag.BoolVar(&options.ignoreerrors, "force", false, "ignore errors, and force working")
	flag.StringVar(&options.logfile, "logfile", "", "log to this file")
	flag.StringVar(&downloader.Options.BackupFolder, "folder", workingDirectory, "backup folder")
	flag.StringVar(&downloader.Options.AlbumID, "album", "", "download only from this album (use google album id)")
	flag.IntVar(&downloader.Options.MaxItems, "max", math.MaxInt32, "max items to download")
	flag.IntVar(&downloader.Options.PageSize, "pagesize", 50, "number of items to download on per API call")
	flag.IntVar(&downloader.Options.Throttle, "throttle", 5, "time, in seconds, to wait between API calls")
	flag.StringVar(&downloader.Options.FolderFormat, "folder-format", filepath.Join("2006", "January"), "time format used for folder paths based on https://golang.org/pkg/time/#Time.Format")
	flag.BoolVar(&downloader.Options.UseFileName, "use-file-name", false, "use file name when uploaded to Google Photos")
	flag.BoolVar(&downloader.Options.IncludeEXIF, "include-exif", false, "retain EXIF metadata on downloaded images. Location information is not included.")
	flag.Float64Var(&downloader.Options.DownloadThrottle, "download-throttle", 0, "rate in KB/sec, to limit downloading of items")
	flag.IntVar(&downloader.Options.ConcurrentDownloads, "concurrent-downloads", 5, "number of concurrent item downloads")
	flag.StringVar(&downloader.Options.CredentialsFile, "credentials-file", "credentials.json", "filepath to where the credentials file can be found")
	flag.StringVar(&downloader.Options.TokenFile, "token-file", "token.json", "filepath to where the token should be stored")

	flag.Parse()
	if options.logfile != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   options.logfile,
			MaxSize:    500, // megabytes
			MaxBackups: 3,
		})
		defer func() {
			if r := recover(); r != nil {
				log.Println(r)
			}
		}()
	}
	err := process(downloader)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
