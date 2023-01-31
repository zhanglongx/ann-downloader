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
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	APP_NAME = "ann-downloader"
	VERSION  = "1.0.2"
)

const (
	// YEARSTBACKWARDS is the years to backwards if not specified
	YEARSTBACKWARDS = 3
)

// Downloader first get all stocks info,
// then download selected symbols and year
// announcements under dir/<stock>.<name>
type Downloader struct {
	// Dir prefix to download
	Dir string

	// Years to lookup, e.g 20xx
	Year []string

	// SkipIfExists
	SkipIfExists bool

	// FilterOutKeyWords
	FilterOutKeyWords []string

	list struct {
		StockList []map[string]string
	}
}

type code struct {
	Stock string
	OrgId string
	// Name removed '*', e.g *ST
	Name string
}

type announcements []map[string]interface{}

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

	optVer := flag.Bool("version", false, "print version")
	optDir := flag.String("dir", defDir, "download directory prefix")
	optNoSkip := flag.Bool("no-skip", false, "no skip if exists")
	optYear := flag.String("year", "", "year to download")

	flag.Parse()

	if *optVer {
		fmt.Printf("%s %s\n", APP_NAME, VERSION)
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

	var years []string
	if *optYear != "" {
		years = make([]string, 1)
		years[0] = *optYear
	}

	dl := Downloader{
		Dir:               defDir,
		Year:              years,
		FilterOutKeyWords: []string{"摘要"},
		SkipIfExists:      !*optNoSkip,
	}

	if err = dl.Init(); err != nil {
		log.Fatal(err)
	}

	if err = dl.Download(optSymbols); err != nil {
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
		DownDir := path.Join(d.Dir, c.Stock+"."+c.Name)

		if _, err := os.Stat(DownDir); os.IsNotExist(err) {
			if err = os.Mkdir(DownDir, os.ModePerm); err != nil {
				return err
			}
		}

		anns, err := d.query(c)
		if err != nil {
			return err
		}

		anns = anns.filterOutKeyWords(d.FilterOutKeyWords).filterYears(d.Year)

		for _, a := range anns {
			adjunctUrl := a["adjunctUrl"].(string)
			title := a["announcementTitle"].(string)

			urlFile := "http://static.cninfo.com.cn/" + adjunctUrl
			pathFile := path.Join(DownDir, title+".pdf")

			if err = d.downFile(urlFile, pathFile); err != nil {
				return err
			}
		}
	}

	return nil
}

// lookUpCode return code and orgId
func (d *Downloader) lookUpCode(contents []string) (ret []code) {

	// TODO: to optimize
	for _, r := range d.list.StockList {
		for _, c := range contents {
			if c == r["code"] || c == r["pinyin"] || c == r["zwjc"] {
				ret = append(ret, code{
					Stock: r["code"],
					OrgId: r["orgId"],
					Name:  strings.Replace(r["zwjc"], "*", "", -1),
				})
			}
		}
	}

	return
}

func (d *Downloader) query(c code) (announcements, error) {

	var rets announcements
	page := 1

	for {
		resp, err := http.PostForm("http://www.cninfo.com.cn/new/hisAnnouncement/query",
			url.Values{
				"pageNum":   []string{strconv.Itoa(page)},
				"pageSize":  []string{"30"},
				"column":    []string{""},
				"tabName":   []string{"fulltext"},
				"plate":     []string{""},
				"stock":     []string{c.Stock + "," + c.OrgId},
				"searchkey": []string{""},
				"secid":     []string{""},
				"category":  []string{"category_ndbg_szsh"},
				"trade":     []string{""},
				"seDate":    []string{""},
				"sortName":  []string{""},
				"sortType":  []string{""},
				"isHLtitle": []string{"true"},
			})

		if err != nil {
			return nil, err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		resp.Body.Close()

		result := make(map[string]interface{}, 1)

		if err = json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		for _, ann := range result["announcements"].([]interface{}) {
			rets = append(rets, ann.(map[string]interface{}))
		}

		if !result["hasMore"].(bool) {
			break
		}

		page++
	}

	return rets, nil
}

// https://golangdocs.com/golang-download-files
func (d *Downloader) downFile(urlFile string, fullname string) error {

	if d.SkipIfExists {
		if _, err := os.Stat(fullname); !os.IsNotExist(err) {
			log.Printf("%s already exists, skip downloading", fullname)
			return nil
		}
	}

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
	resp, err := client.Get(urlFile)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)

	return err
}

func (a announcements) filterOutKeyWords(keyWords []string) (rets announcements) {

LABELKEYWORDSLOOP:
	for _, ann := range a {
		title := ann["announcementTitle"].(string)

		for _, no := range keyWords {
			if strings.Contains(title, no) {
				continue LABELKEYWORDSLOOP
			}
		}

		rets = append(rets, ann)
	}

	return
}

func (a announcements) filterYears(years []string) (rets announcements) {

	toSort := make(map[string]announcements, 0)
	reg := regexp.MustCompile(`20\d\d`)

	for _, ann := range a {
		title := ann["announcementTitle"].(string)

		// FIXME: the most tricky one is (20xx amend), so simply
		// find by leftmost for now
		year := reg.FindString(title)
		if year == "" {
			continue
		}

		toSort[year] = append(toSort[year], ann)
	}

	if years == nil {
		keys := make([]string, 0, len(toSort))

		for k := range toSort {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for i := 0; i < YEARSTBACKWARDS && i < len(keys); i++ {
			years = append(years, keys[len(keys)-1-i])
		}
	}

	for _, y := range years {
		if toSort[y] == nil {
			log.Printf("%s announcement cannot be found", y)
			continue
		}

		rets = append(rets, toSort[y]...)
	}

	return
}
