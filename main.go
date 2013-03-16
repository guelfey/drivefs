package main

import (
	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/drive/v2"
	"flag"
	"fmt"
	"github.com/hanwen/go-fuse/fuse"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"path"
	"syscall"
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

var (
	debugApi  = flag.Bool("debug-api", false, "print Drive API debugging output")
	debugFuse = flag.Bool("debug-fuse", false, "print FUSE debugging output")
	doInit    = flag.Bool("init", false, "retrieve a new token")
	tokenFile = flag.String("tokenfile", getTokenFile(), "path to the token file")
)

type debugTransport struct {
	tr http.RoundTripper
}

func (d debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	buf, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Println("failed to dump request:", err)
	} else {
		log.Printf("sending request: %s\n", buf)
	}
	resp, err := d.tr.RoundTrip(req)
	if err != nil {
		log.Println("got error:", err)
	} else {
		buf, err = httputil.DumpResponse(resp, true)
		if err != nil {
			log.Println("failed to dump response", err)
		} else {
			log.Printf("got response: %s\n", buf)
		}
	}
	return resp, err
}

func getTokenFile() string {
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
	cache := oauth.CacheFile(*tokenFile)
	tok, err := cache.Token()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read token:", err)
		fmt.Fprintln(os.Stderr, "Did you run drivefs -init?")
		os.Exit(1)
	} else {
		transport.Token = tok
	}
	srv, err = drive.New(transport.Client())
	if err != nil {
		log.Fatalln("Failed to create drive service:", err)
	}
}

func getToken() {
	var code string
	if _, err := os.Stat(path.Dir(*tokenFile)); os.IsNotExist(err) {
		if err = os.MkdirAll(path.Dir(*tokenFile), 0755); err != nil {
			log.Fatalln("Failed to create cache directory:", err)
		}
	}
	cache := oauth.CacheFile(*tokenFile)
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

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: drivefs [ options ... ] mountpoint")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	transport.Config = oauthConf
	flag.Usage = usage
	flag.Parse()
	if *debugApi {
		transport.Transport = debugTransport{http.DefaultTransport}
	}
	if *doInit {
		getToken()
		return
	}
	if flag.NArg() < 1 {
		usage()
	}
	connect()
	fs.root = &dirNode{}
	fs.uid = uint32(os.Getuid())
	fs.gid = uint32(os.Getgid())
	fsc := fuse.NewFileSystemConnector(&fs, nil)
	ms := fuse.NewMountState(fsc)
	err := ms.Mount(flag.Arg(0), &fuse.MountOptions{Name: "drivefs"})
	if err != nil {
		log.Fatalln("Failed to mount file system:", err)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for {
			<-c
			ms.Unmount()
		}
	}()
	ms.Debug = *debugFuse
	ms.Loop()
}
