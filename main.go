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

	"github.com/dtylman/gitmoo-goog/downloader"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	photoslibrary "google.golang.org/api/photoslibrary/v1"
	"gopkg.in/natefinch/lumberjack.v2"
)

//Version is the version number
const Version = "0.23"

var options struct {
	loop         bool
	logfile      string
	ignoreerrors bool
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
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

func process() error {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Println("Enable photos API here: https://developers.google.com/photos/library/guides/get-started#enable-the-api")
		return fmt.Errorf("Unable to read client secret file: %v", err)
	}

	//request photos readonly access
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/photoslibrary.readonly")
	if err != nil {
		return fmt.Errorf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)
	log.Printf("Connecting ...")
	srv, err := photoslibrary.New(client)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Sheets client: %v", err)
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
	log.Println("This is gitmoo-goog ver", Version)
	flag.BoolVar(&options.loop, "loop", false, "loops forever (use as daemon)")
	flag.BoolVar(&options.ignoreerrors, "force", false, "ignore errors, and force working")
	flag.StringVar(&options.logfile, "logfile", "", "log to this file")
	flag.StringVar(&downloader.Options.BackupFolder, "folder", "", "backup folder")
	flag.StringVar(&downloader.Options.AlbumID, "album", "", "download only from this album (use google album id)")
	flag.IntVar(&downloader.Options.MaxItems, "max", math.MaxInt32, "max items to download")
	flag.IntVar(&downloader.Options.PageSize, "pagesize", 50, "number of items to download on per API call")
	flag.IntVar(&downloader.Options.Throttle, "throttle", 5, "Time, in seconds, to wait between API calls")

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
	err := process()
	if err != nil {
		log.Println(err)
	}
}
