package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// A Download process download in this app.
type Download struct {
	Client   *Client `ini:"-"`
	IDOrList string  `ini:"-"`
	Path     string
	Naming   Naming  `ini:",omitempty"`
	Metadata string  `ini:",omitempty"`
}

// A Naming save naming pattern of downloaded files.
type Naming struct {
	SingleFile   string
	MultipleFile string
	Folder       string
}

// ArtistData save the data of a artist.
type ArtistData struct {
	ID       string `tag:"artist.id"`       // `href="/member.php?id=(\d+?)" class="tab-profile"`
	Username string `tag:"artist.username"` // `href="/stacc/(.+?)" class="tab-feed"`
	Nickname string `tag:"artist.nickname"` // `<span class="user-name">(.+?)</span>`
}

// A WorkType resolve the type of a work.
type WorkType uint8

const (
	Illust WorkType = iota
	Ugoira
	Manga
)

// A WorkData save the data of a work.
type WorkData struct {
	ID        string     `tag:"work.id"`
	Name      string     `tag:"work.name"`
	Time      time.Time  `tag:"work.time"`
	PageCount uint64     `tag:"work.page_count"`
	Tools     []string   `tag:"work.tools"`
	Series    string     `tag:"work.series"`
	Caption   string     `tag:"work.caption" naming:"-"`
	Tags      []string   `tag:"work.tags"`
	Type      WorkType   `tag:"work.type"`
	Pages     []PageData `tag:"work.pages" naming:"-"`
	Thumb     string     `tag:"work.thumb" naming:"-"`
}

// A PageData save the data of a page of a work.
type PageData struct {
	Page     uint64 `tag:"page"`
	Width    uint64 `tag:"width"`
	Height   uint64 `tag:"height"`
	Filename string `tag:"filename"`
	ImageURL string `tag:"url" naming:"-"`
}

// Do run download process in this app.
func (d *Download) Do() (err error) {
	var (
		resp             *http.Response
		isLoggedIn, isID bool
	)
	
	// Check that pixiv is already logged or not.
	if resp, err = d.Client.Get(PixivHomeURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if isLoggedIn, err = checkIsLoggedIn(resp, d,
		"request status is not OK when checking "+
				"that not login yet or not"); err != nil {
		return err
	} else if !isLoggedIn {
		return throw(d, "not logged in yet")
	}
	
	// Decide Download.IDOrList is ID or list and run corresponding function.
	if isID, err = d.isIDOrList(); err != nil {
		return err
	}
	if isID {
		err = d.downloadFromID()
	} else {
		err = d.downloadFromList()
	}
	
	return err
}

// isIDOrList decide Download.IDOrList is ID or list.
func (d *Download) isIDOrList() (isID bool, err error) {
	isID = true
	for _, c := range d.IDOrList {
		if c < '0' || c > '9' {
			isID = false
		}
	}
	if !isID {
		
	}
	return isID, nil
}

// downloadFromID download work from given Pixiv work ID.
func (d *Download) downloadFromID() (err error) {
	var (
		artistData = new(ArtistData)
		workData   = new(WorkData)
	)
	// Before calling Download.download, workData should include ID value.
	workData.ID = d.IDOrList
	if err = d.download(artistData, workData); err != nil {
		return err
	}
	return nil
}

// downloadFromList download works from given list that include Pixiv work IDs.
func (d *Download) downloadFromList() (err error) {
	// TODO: download from list.
	var (
	// artistData = new(ArtistData)
	// workData   = new(WorkData)
	)
	// Set artistData and workData from list file and download,
	// it should include ID value in workData.
	return nil
}

// download get exile data of work and artist and download work.
func (d *Download) download(artistData *ArtistData, workData *WorkData) (err error) {
	var (
		resp *http.Response
		body string
	)
	
	// TODO: process of get exile data of work and artist.
	
	// Get response body of the work.
	if resp, err = d.Client.Get(fmt.Sprintf(PixivWorkURL, workData.ID)); err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return throw(d, "request status is not OK when getting work page")
	}
	if body, err = getResponseBody(resp); err != nil {
		return err
	}
	
	// Get data of work and artist.
	d.getArtistData(body, artistData)
	if err = d.getWorkData(body, workData); err != nil {
		return err
	}
	
	// Download work(s).
	// TODO: not complete yet.
	var bodyBytes []byte
	for _, page := range workData.Pages {
		if resp, err = d.Client.Get(page.ImageURL); err != nil {
			return err
		}
		bodyBytes, err = ioutil.ReadAll(resp.Body)
		ioutil.WriteFile(page.Filename, bodyBytes, 0644)
		resp.Body.Close()
	}
	
	return nil
}

// getArtistData get artist data from response body of a work.
func (d *Download) getArtistData(body string, artistData *ArtistData) {
	
	// TODO: process of get exile data.
	
	// get artist ID, username, nickname.
	artistData.ID = regexp.MustCompile(
		`href="/member.php\?id=(\d+?)" class="tab-profile"`).
			FindStringSubmatch(body)[1]
	artistData.Username = regexp.MustCompile(
		`href="/stacc/(.+?)" class="tab-feed"`).FindStringSubmatch(body)[1]
	artistData.Nickname = regexp.MustCompile(
		`<span class="user-name">(.+?)</span>`).FindStringSubmatch(body)[1]
	
	// fmt.Printf("artistData:\n")
	// fmt.Printf("ID->%v\n", artistData.ID)
	// fmt.Printf("Username->%v\n", artistData.Username)
	// fmt.Printf("Nickname->%v\n", artistData.Nickname)
}

// getArtistData get work data from response body of a work.
func (d *Download) getWorkData(body string, workData *WorkData) (err error) {
	
	// TODO: process of get exile data.
	
	var (
		singleMatch = func(body, str string) string {
			return regexp.MustCompile(str).FindStringSubmatch(body)[1]
		}
		meta, tags, workType, thumbURL string
		seriesMatch, captionMatch      []string
		metaMatch, tagsMatch           [][]string
		bodyBytes                      []byte
		resp                           *http.Response
	)
	
	// fmt.Printf("workData:\n")
	// fmt.Printf("ID->%v\n", workData.ID)
	
	// Get work name.
	workData.Name = singleMatch(body, `<h1 class="title">(.+?)</h1>`)
	// fmt.Printf("Name->%v\n", workData.Name)
	
	// Get work meta that include time, page count or width / height, and tools.
	meta = singleMatch(body, `<ul class="meta">(.+?)</ul>(<div `+
			`class="_illust-series-title">(.+?)</div>)?<h1 class="title">`)
	metaMatch = regexp.MustCompile(`<li>(<ul class="tools">` +
			`(.+?)</ul>)?(.+?)?</li>`).FindAllStringSubmatch(meta, -1)
	
	// Get time from meta.
	if workData.Time, err = time.Parse("2006年1月2日 15:04 MST",
		metaMatch[0][3]+" JST"); err != nil {
		return err
	}
	// fmt.Printf("Time->%v\n", workData.Time)
	
	// Get page count or width / height from meta.
	if strings.Index(metaMatch[1][3], "×") >= 0 {
		// When width / height case, page count is 1.
		workData.PageCount = 1
		workData.Pages = make([]PageData, workData.PageCount)
		var size = strings.Split(metaMatch[1][3], "×")
		if workData.Pages[0].Width, err = strconv.ParseUint(
			size[0], 10, 64); err != nil {
			return err
		}
		if workData.Pages[0].Height, err = strconv.ParseUint(
			size[1], 10, 64); err != nil {
			return err
		}
		// fmt.Printf("Pages[0].Width->%v\n", workData.Pages[0].Width)
		// fmt.Printf("Pages[0].Height->%v\n", workData.Pages[0].Height)
	} else if strings.Index(metaMatch[1][3], "P") >= 0 {
		// When page count case, width / height can only get from file.
		if workData.PageCount, err = strconv.ParseUint(singleMatch(
			metaMatch[1][3], `^.* (\d+)P$`), 10, 64); err != nil {
			return err
		}
		workData.Pages = make([]PageData, workData.PageCount)
	}
	// fmt.Printf("PageCount->%v\n", workData.PageCount)
	
	// Get tools from meta.
	if len(metaMatch) == 3 {
		var toolsMatch = regexp.MustCompile(`<li>(.+?)</li>`).
				FindAllStringSubmatch(metaMatch[2][2], -1)
		workData.Tools = make([]string, len(toolsMatch))
		for i, tool := range toolsMatch {
			workData.Tools[i] = tool[1]
		}
	}
	// fmt.Printf("Tools->%v\n", workData.Tools)
	
	// Get work series.
	seriesMatch = regexp.MustCompile(`<a class="_illust-series-title-text` +
			`" href=".+?">(.+?)</a>`).FindStringSubmatch(body)
	if len(seriesMatch) > 1 {
		workData.Series = seriesMatch[1]
	}
	// fmt.Printf("Series->%v\n", workData.Series)
	
	// Get work caption.
	captionMatch = regexp.MustCompile(
		`<p class="caption">(.+?)</p>`).FindStringSubmatch(body)
	if len(captionMatch) > 1 {
		workData.Caption = captionMatch[1]
	}
	// fmt.Printf("Caption->%v\n", workData.Caption)
	
	// Get work tags.
	tags = singleMatch(body, `<span class="tags-container">`+
			`(.+?)</span><script id="template-work-tags"`)
	tagsMatch = regexp.MustCompile(`class="text">(.+?)</a>`).
			FindAllStringSubmatch(tags, -1)
	workData.Tags = make([]string, len(tagsMatch))
	for i, tag := range tagsMatch {
		workData.Tags[i] = tag[1]
	}
	// fmt.Printf("Tags->%v\n", workData.Tags)
	
	// Get work type.
	workType = singleMatch(body,
		`class="(.+?)"><div class="_layout-thumbnail">`)
	if strings.Index(workType, "ugoku-illust") >= 0 {
		workData.Type = Ugoira
	} else if strings.Index(workType, "manga") >= 0 {
		workData.Type = Manga
	} else {
		workData.Type = Illust
	}
	// fmt.Printf("Type->%v\n", workData.Type)
	
	// Get work thumbnail in base64 form.
	thumbURL = singleMatch(body,
		`class="bookmark_modal_thumbnail" data-src="(.+?)"`)
	if resp, err = d.Client.Get(thumbURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if bodyBytes, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}
	workData.Thumb = base64.StdEncoding.EncodeToString(bodyBytes)
	
	// Get URL and filename of each image of work.
	if workData.PageCount == 1 {
		workData.Pages[0].Page = 0
		if workData.Type != Ugoira {
			workData.Pages[0].ImageURL = singleMatch(body,
				`data-src="(.+?)" class="original-image"`)
		}
		workData.Pages[0].Filename = path.Base(workData.Pages[0].ImageURL)
		// fmt.Printf("Pages[0].ImageURL->%v\n", workData.Pages[0].ImageURL)
		// fmt.Printf("Pages[0].Filename->%v\n", workData.Pages[0].Filename)
	} else if workData.PageCount > 1 {
		for i := uint64(0); i < workData.PageCount; i++ {
			workData.Pages[i].Page = i
			if resp, err = d.Client.Get(fmt.Sprintf(
				PixivMangaURL, workData.ID, i)); err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return throw(d,
					"request status is not OK when getting manga page")
			}
			if body, err = getResponseBody(resp); err != nil {
				return err
			}
			workData.Pages[i].ImageURL = singleMatch(body, `src="(.+?)"`)
			workData.Pages[i].Filename = path.Base(workData.Pages[i].ImageURL)
			resp.Body.Close()
			// fmt.Printf("Pages[%d].ImageURL->%v\n", i, workData.Pages[i].ImageURL)
			// fmt.Printf("Pages[%d].Filename->%v\n", i, workData.Pages[i].Filename)
		}
	}
	
	return nil
}
