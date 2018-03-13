package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Download struct {
	Client   *Client `ini:"-"`
	IDOrList string  `ini:"-"`
	Path     string
	Naming   Naming  `ini:",omitempty"`
	Metadata string  `ini:",omitempty"`
}

type Naming struct {
	SingleFile   string
	MultipleFile string
	Folder       string
}

type ArtistData struct {
	ID       string `tag:"artist.id"`       // `href="/member.php?id=(\d+?)" class="tab-profile"`
	Username string `tag:"artist.username"` // `href="/stacc/(.+?)" class="tab-feed"`
	Nickname string `tag:"artist.nickname"` // `<span class="user-name">(.+?)</span>`
}

type WorkType int

const (
	Illust WorkType = iota
	Ugoira
	Manga
)

type WorkData struct {
	ID      string     `tag:"work.id"`
	Name    string     `tag:"work.name"`
	Time    time.Time  `tag:"work.time"`
	Pages   int        `tag:"work.pages"`
	Tools   []string   `tag:"work.tool"`
	Series  string     `tag:"work.series"`
	Caption string     `tag:"work.caption" naming:"-"`
	Tags    []string   `tag:"work.tags"`
	Type    WorkType   `tag:"work.type"`
	Page    []PageData `tag:"work.caption" naming:"-"`
	Thumb   string     `tag:"work.thumb" naming:"-"`
}

type PageData struct {
	Page     int    `tag:"page"`
	Width    int    `tag:"width"`
	Height   int    `tag:"height"`
	Filename string `tag:"filename"`
	ImageURL string `tag:"url" naming:"-"`
}

func (d *Download) Do() (err error) {
	var (
		resp             *http.Response
		isLoggedIn, isID bool
	)
	
	// Check that pixiv is already logged or not
	if resp, err = d.Client.Get(PixivHomeURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if isLoggedIn, err = checkIsLoggedIn(resp,
		"download: request status is not OK when checking that not login yet or not"); err != nil {
		return err
	} else if !isLoggedIn {
		return errors.New("download: not logged in yet")
	}
	
	if isID, err = d.IsIDOrList(); err != nil {
		return err
	}
	if isID {
		err = d.downloadFromID()
	} else {
		err = d.downloadFromList()
	}
	
	return err
}

func (d *Download) IsIDOrList() (isID bool, err error) {
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

func (d *Download) downloadFromID() (err error) {
	var (
		artistData = new(ArtistData)
		workData   = new(WorkData)
	)
	
	workData.ID = d.IDOrList
	if err = d.download(artistData, workData); err != nil {
		return err
	}
	
	return nil
}

func (d *Download) downloadFromList() (err error) {
	
	return nil
}

func (d *Download) getArtistData(body string, artistData *ArtistData) {
	artistData.ID = regexp.MustCompile(`href="/member.php\?id=(\d+?)" class="tab-profile"`).FindStringSubmatch(body)[1]
	artistData.Username = regexp.MustCompile(`href="/stacc/(.+?)" class="tab-feed"`).FindStringSubmatch(body)[1]
	artistData.Nickname = regexp.MustCompile(`<span class="user-name">(.+?)</span>`).FindStringSubmatch(body)[1]
	
	fmt.Printf("artistData:\n")
	fmt.Printf("ID->%v\n", artistData.ID)
	fmt.Printf("Username->%v\n", artistData.Username)
	fmt.Printf("Nickname->%v\n", artistData.Nickname)
}

func (d *Download) getWorkData(body string, workData *WorkData) (err error) {
	var (
		singleMatch = func(body, str string) string {
			return regexp.MustCompile(str).FindStringSubmatch(body)[1]
		}
		meta, tags, wType, thumbURL string
		seriesMatch, captionMatch   []string
		metaMatch, tagsMatch        [][]string
		bodyBytes                   []byte
		resp                        *http.Response
	)
	fmt.Printf("workData:\n")
	fmt.Printf("ID->%v\n", workData.ID)
	
	workData.Name = singleMatch(body, `<h1 class="title">(.+?)</h1>`)
	fmt.Printf("Name->%v\n", workData.Name)
	
	meta = singleMatch(body, `<ul class="meta">(.+?)</ul>(<div class="_illust-series-title">(.+?)</div>)?<h1 class="title">`)
	metaMatch = regexp.MustCompile(`<li>(<ul class="tools">(.+?)</ul>)?(.+?)?</li>`).FindAllStringSubmatch(meta, -1)
	
	if workData.Time, err = time.Parse("2006年1月2日 15:04 MST", metaMatch[0][3]+" JST"); err != nil {
		return err
	}
	fmt.Printf("Time->%v\n", workData.Time)
	
	if strings.Index(metaMatch[1][3], "×") >= 0 {
		workData.Pages = 1
		workData.Page = make([]PageData, workData.Pages)
		var size = strings.Split(metaMatch[1][3], "×")
		if workData.Page[0].Width, err = strconv.Atoi(size[0]); err != nil {
			return err
		}
		if workData.Page[0].Height, err = strconv.Atoi(size[1]); err != nil {
			return err
		}
		fmt.Printf("Page[0].Width->%v\n", workData.Page[0].Width)
		fmt.Printf("Page[0].Height->%v\n", workData.Page[0].Height)
	} else if strings.Index(metaMatch[1][3], "P") >= 0 {
		if workData.Pages, err = strconv.Atoi(singleMatch(metaMatch[1][3], `^.* (\d+)P$`)); err != nil {
			return err
		}
		workData.Page = make([]PageData, workData.Pages)
	}
	fmt.Printf("Pages->%v\n", workData.Pages)
	
	if len(metaMatch) == 3 {
		var toolsMatch = regexp.MustCompile(`<li>(.+?)</li>`).FindAllStringSubmatch(metaMatch[2][2], -1)
		workData.Tools = make([]string, len(toolsMatch))
		for i, tool := range toolsMatch {
			workData.Tools[i] = tool[1]
		}
	}
	fmt.Printf("Tools->%v\n", workData.Tools)
	
	seriesMatch = regexp.MustCompile(`<a class="_illust-series-title-text" href=".+?">(.+?)</a>`).FindStringSubmatch(body)
	if len(seriesMatch) > 1 {
		workData.Series = seriesMatch[1]
	}
	fmt.Printf("Series->%v\n", workData.Series)
	
	captionMatch = regexp.MustCompile(`<p class="caption">(.+?)</p>`).FindStringSubmatch(body)
	if len(captionMatch) > 1 {
		workData.Caption = captionMatch[1]
	}
	fmt.Printf("Caption->%v\n", workData.Caption)
	
	tags = singleMatch(body, `<span class="tags-container">(.+?)</span><script id="template-work-tags"`)
	tagsMatch = regexp.MustCompile(`class="text">(.+?)</a>`).FindAllStringSubmatch(tags, -1)
	workData.Tags = make([]string, len(tagsMatch))
	for i, tag := range tagsMatch {
		workData.Tags[i] = tag[1]
	}
	fmt.Printf("Tags->%v\n", workData.Tags)
	
	wType = singleMatch(body, `class="(.+?)"><div class="_layout-thumbnail">`)
	if strings.Index(wType, "ugoku-illust") >= 0 {
		workData.Type = Ugoira
	} else if strings.Index(wType, "manga") >= 0 {
		workData.Type = Manga
	} else {
		workData.Type = Illust
	}
	fmt.Printf("Type->%v\n", workData.Type)
	
	thumbURL = singleMatch(body, `class="bookmark_modal_thumbnail" data-src="(.+?)"`)
	if resp, err = d.Client.Get(thumbURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if bodyBytes, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}
	workData.Thumb = base64.StdEncoding.EncodeToString(bodyBytes)
	
	if workData.Pages == 1 {
		workData.Page[0].Page = 0
		if workData.Type != Ugoira {
			workData.Page[0].ImageURL = singleMatch(body, `data-src="(.+?)" class="original-image"`)
		}
		workData.Page[0].Filename = path.Base(workData.Page[0].ImageURL)
		fmt.Printf("Page[0].ImageURL->%v\n", workData.Page[0].ImageURL)
		fmt.Printf("Page[0].Filename->%v\n", workData.Page[0].Filename)
	} else if workData.Pages > 1 {
		for i := 0; i < workData.Pages; i++ {
			workData.Page[i].Page = i
			if resp, err = d.Client.Get(fmt.Sprintf(PixivMangaURL, workData.ID, i)); err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return errors.New("download: request status is not OK when getting manga page")
			}
			if body, err = getResponseBody(resp); err != nil {
				return err
			}
			workData.Page[i].ImageURL = singleMatch(body, `src="(.+?)"`)
			workData.Page[i].Filename = path.Base(workData.Page[i].ImageURL)
			resp.Body.Close()
			fmt.Printf("Page[%d].ImageURL->%v\n", i, workData.Page[i].ImageURL)
			fmt.Printf("Page[%d].Filename->%v\n", i, workData.Page[i].Filename)
		}
	}
	
	return nil
}

func (d *Download) download(artistData *ArtistData, workData *WorkData) (err error) {
	var (
		resp *http.Response
		body string
	)
	if resp, err = d.Client.Get(fmt.Sprintf(PixivWorkURL, workData.ID)); err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("download: request status is not OK when getting work page")
	}
	if body, err = getResponseBody(resp); err != nil {
		return err
	}
	
	d.getArtistData(body, artistData)
	if err = d.getWorkData(body, workData); err != nil {
		return err
	}
	
	var bodyBytes []byte
	for _, page := range workData.Page {
		if resp, err = d.Client.Get(page.ImageURL); err != nil {
			return err
		}
		bodyBytes, err = ioutil.ReadAll(resp.Body)
		ioutil.WriteFile(page.Filename, bodyBytes, 0644)
		resp.Body.Close()
	}
	
	return nil
}
