package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const (
	APP_NAME = "ann-downloader"
	VERSION  = "0.9.0"

	YEARS = 3
)

type Downloader struct {
	Dir string

	Year []int

	// TODO
	SkipExists bool

	list struct {
		StockList []map[string]string
	}
}

type Code struct {
	Code  string
	OrgId string
	Name  string
}

func main() {

	defDir, err := defaultDir()
	if err != nil {
		log.Printf("HOME not exist, using current directory as default")
		if defDir, err = os.Getwd(); err != nil {
			log.Fatal(err)
		}
	}

	if _, err := os.Stat(defDir); os.IsNotExist(err) {
		log.Printf("%s not exist, using current directory as default", defDir)
		if defDir, err = os.Getwd(); err != nil {
			log.Fatal(err)
		}
	}

	if ok, err := isDirectory(defDir); !ok || err != nil {
		log.Printf("%s is not a directory, using current directory as default", defDir)
		if defDir, err = os.Getwd(); err != nil {
			log.Fatal(err)
		}
	}

	defYear, _, _ := time.Now().Date()

	optVer := flag.Bool("version", false, "print version")
	optDir := flag.String("dir", defDir, "download directory")

	flag.Parse()

	if *optVer {
		fmt.Printf("%s %s", APP_NAME, VERSION)
		os.Exit(0)
	}

	if _, err = os.Stat(*optDir); os.IsNotExist(err) {
		log.Fatalf("%s not exists", *optDir)
	}

	optSymbols := flag.Args()
	if len(optSymbols) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	dl := Downloader{
		Dir:  defDir,
		Year: []int{defYear - 1, defYear - 2},
	}

	if err = dl.Init(); err != nil {
		log.Fatal(err)
	}
}

func defaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(home, "Dropbox", "Personal", "年报"), nil
}

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), err
}

func (d *Downloader) Init() error {

	resp, err := http.Get("http://www.cninfo.com.cn/new/data/szse_stock.json")
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("init with ret code error")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(body, &d.list); err != nil {
		return err
	}

	return nil
}

func (d *Downloader) Download(symbols []string) error {

	codes := d.lookUpCode(symbols)
	if len(codes) == 0 {
		return errors.New("lookup symbol(s) failed")
	}

	for _, c := range codes {
		DownDir := path.Join(d.Dir, c.Code+"."+c.Name)

		if _, err := os.Stat(DownDir); os.IsNotExist(err) {
			if err = os.Mkdir(DownDir, os.ModePerm); err != nil {
				return err
			}
		}
	}

	return nil
}

// lookUpCode return code and orgId
func (d *Downloader) lookUpCode(contents []string) (ret []Code) {

	// TODO: to optimize
	for _, r := range d.list.StockList {
		for _, c := range contents {
			if c == r["code"] || c == r["pinyin"] || c == r["zwjc"] {
				ret = append(ret, Code{
					Code:  r["code"],
					OrgId: r["orgId"],
					Name:  strings.Replace(r["zwjc"], "*", "", -1),
				})
			}
		}
	}

	return
}

// https://golangdocs.com/golang-download-files
func (d *Downloader) downFile(url string, fullname string) error {

	// Create blank file
	file, err := os.Create(fullname)
	if err != nil {
		return err
	}

	defer file.Close()

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	// Put content on file
	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)

	return err
}
