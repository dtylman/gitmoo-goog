# gitmoo-goog

A [Google Photos](http://photos.google.com/) backup tool.

`gitmoo-goog` tool uses [Google Photos API](https://developers.google.com/photos/library/guides/get-started#enable-the-api) to continously download all photos on a local device.

It can be used as a daemon to keep in sync with a google-photos account.

[![CircleCI](https://circleci.com/gh/dtylman/gitmoo-goog.svg?style=svg)](https://circleci.com/gh/dtylman/gitmoo-goog)

## Downloading and Installing:

Download:
* [Linux](https://github.com/dtylman/gitmoo-goog/releases/download/0.21/gitmoo-goog.bz2) 
* [Windows](https://github.com/dtylman/gitmoo-goog/releases/download/0.21/gitmoo-goog.zip)
* [MacOS](https://github.com/dtylman/gitmoo-goog/releases/download/0.21/gitmoo-goog.gz)

Unzip and run, there are no other dependencies.

## Usage:

### Enable Google-Photos API:

(There is more than one way to do the following):
* Go to [API](https://developers.google.com/photos/library/guides/get-started#enable-the-api)
* Click on `Enable the Google Photos API` button
* Create a new project, name it whatever you like.
* Write `gitmoo-goog` in the `product name` field, (can be whatever you like)
* On the `when are you calling from`, choose `Other`, click `Create`.
* Download the client configuration.

### Configure gitmoo-goog:

* Copy the downloaded `credentials.json` to the same folder with `gitmoo-goog` .
* run `gitmoo-goog`, and follow the provided link:
* Sign in, and click `Allow`
* Copy the `token` and paste it in the cli.
* `gitmoo-goog` will authorize and start downloading content. 
```
$ ./gitmoo-goog
2018/09/12 10:18:07 This is gitmoo-goog ver 0.1
Go to the following link in your browser then type the authorization code:
https://accounts.google.com/o/oauth2/auth?access_type=...
4/WACqgFeX5OTB8X4LWd5i2TFH....
Saving credential file to: token.json
2018/09/12 10:20:07 Connecting ...
2018/09/12 10:20:07 Processed: 0, Downloaded: 0, Errors: 0, Total Size: 0 B, Waiting 5s
```


This is probably not what you want, hit `crt-c` to stop it.

### Usage:

```
Usage of ./gitmoo-goog:
  - album
        download only from this album (use google album id)
  -folder string
        backup folder (default current working directory)
  -force
        ignore errors, and force working
  -logfile string
        log to this file
  -credentials-file string
        filepath to where the credentials file can be found (default 'credentials.json')
  -token-file string
        filepath to where the token should be stored (default 'token.json')
  -loop
        loops forever (use as daemon)
  -max int
        max items to download (default 2147483647)
  -pagesize int
        number of items to download on per API call (default 50)
  -throttle int
        Time, in seconds, to wait between API calls (default 5)
  -folder-format string
        Time format used for folder paths based on https://golang.org/pkg/time/#Time.Format (default "2016/Janurary")
  -use-file-name
        Use file name when uploaded to Google Photos (default off)
  -include-exif
        Retain EXIF metadata on downloaded images. Location information is not included because Google does not include it. (default off)
  -download-throttle
        Rate in KB/sec, to limit downloading of items (default off)
  -concurrent-downloads
        Number of concurrent item downloads (default 5)
```

On Linux, running the following is a good practice:

```
$ ./gitmoo-goog -folder archive -logfile gitmoo.log -use-file-name -include-exif -loop -throttle 45 &
```

This will start the process in background, making an API call every 45 seconds, looping forever on all items and saving them to `{pwd}/archive`. All images will be downloaded with a filename and metadata as close to original as Google offers through the api.

Logfile will be saved as `gitmoo.log`.

#### Naming

Files are created as follows:

`[folder][year][month][day]_[hash].json` and `.jpg`. The `json` file holds the metadata from `google-photos`. 

## Docker (Linux only)

You can run gitmoo-goog in Docker. At the moment you have to build the image yourself. After cloning the repo run:

```
$ docker build -t dtylman/gitmoo-goog:latest .
```

Now run gitmoo-goo in Docker:

```
$ docker run -v $(pwd):/app --user=$(id -u):$(id -g) dtylman/gitmoo-goog:latest
```

Replace `$(pwd)` with the location of your storage directory on your computer.
Within the storage directory gitmoo-goog expects the `credentials.json` and will place all the downloaded files.

The part `--user=$(id -u):$(id -g)` ensures that the downloaded files are owned by the user launching the container.

Configuring additional settings is possible by adding command arguments like so:
```
$ docker run -v $(pwd):/app --user=$(id -u):$(id -g) dtylman/gitmoo-goog:latest -loop -throttle 45
```
