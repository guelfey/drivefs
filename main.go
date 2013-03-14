package main

import (
	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/drive/v2"
	"flag"
	"fmt"
	"github.com/hanwen/go-fuse/fuse"
	"log"
	"os"
	"path"
)

var oauthConf = &oauth.Config{
	ClientId:     "391165590784.apps.googleusercontent.com",
	ClientSecret: "FPe6dekrpXuM3RUfg4A6lAvm",
	Scope:        drive.DriveScope,
	AuthURL:      "https://accounts.google.com/o/oauth2/auth",
	TokenURL:     "https://accounts.google.com/o/oauth2/token",
}

var (
	fs        Filesystem
	srv       *drive.Service
	transport oauth.Transport
)

func getCacheFile() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home := os.Getenv("HOME")
		if home == "" {
			log.Fatalln("Failed to determine token location (neither HOME nor" +
				" XDG_DATA_HOME are set)")
		}
		return home + "/.local/share/drivefs/token"
	}
	return dataHome + "/drivefs/token"
}

func connect() {
	cacheFile := getCacheFile()
	cache := oauth.CacheFile(cacheFile)
	tok, err := cache.Token()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Did you run drivefs -init?")
		os.Exit(1)
	} else {
		transport.Token = tok
	}
	srv, err = drive.New(transport.Client())
	if err != nil {
		log.Fatalln("Failed to create drive service:", err)
	}
	transport.Refresh()
}

func getToken() {
	var code string
	cacheFile := getCacheFile()
	if _, err := os.Stat(path.Dir(cacheFile)); os.IsNotExist(err) {
		if err = os.MkdirAll(path.Dir(cacheFile), 0755); err != nil {
			log.Fatalln("Failed to create cache directory:", err)
		}
	}
	cache := oauth.CacheFile(cacheFile)
	url := transport.AuthCodeURL("")
	fmt.Println("Visit this URL, log in with your google account and enter the authorization code here:")
	fmt.Println(url)
	fmt.Scanln(&code)
	tok, err := transport.Exchange(code)
	if err != nil {
		log.Fatalln("Failed to exchange token:", err)
	}
	err = cache.PutToken(tok)
	if err != nil {
		log.Fatalln("Failed to save token:", err)
	}
}

func main() {
	transport.Config = oauthConf
	doInit := flag.Bool("init", false, "Get the oauth token")
	flag.Parse()
	if *doInit {
		getToken()
		return
	}
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: drivefs MOUNTPOINT")
		os.Exit(1)
	}
	connect()
	fs.root = &dirNode{}
	state, _, err := fuse.MountNodeFileSystem(flag.Arg(0), &fs, nil)
	if err != nil {
		log.Fatal(err)
	}
	state.Loop()
}
