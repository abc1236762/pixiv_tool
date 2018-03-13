package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	
	"github.com/juju/persistent-cookiejar"
)

const (
	PixivHomeURL   = "https://www.pixiv.net/"
	PixivLoginURL  = "https://accounts.pixiv.net/login?lang=ja&source=pc&view_type=page&ref=wwwtop_accounts_index"
	PixivLogoutURL = "https://www.pixiv.net/logout.php?return_to=%2F"
	PixivWorkURL   = "https://www.pixiv.net/member_illust.php?mode=medium&illust_id=%s"
	PixivMangaURL  = "https://www.pixiv.net/member_illust.php?mode=manga_big&illust_id=%s&page=%d"
	UserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:58.0.2) Gecko/20100101 Firefox/58.0.2"
)

type PixivError struct {
	Prefix string
	Msg string
}

func (p *PixivError) Error() string {
	return p.Prefix + ": " + p.Msg
}

func main() {
	var (
		client *Client
		err    error
	)
	
	defer func() {
		var a = recover()
		if a != nil {
			fmt.Println("error:", a)
			os.Exit(1)
		}
	}()
	
	if len(os.Args) == 1 {
		var pixiv = Pixiv{}
		
		pixiv.Init()
		return
	}
	
	// test
	if client, err = setHttpClient(); err != nil {
		panic(err)
	}
	if strings.TrimSpace(os.Args[1]) == "login" {
		var login = Login{
			Client:   client,
			Username: strings.TrimSpace(os.Args[2]),
			Password: strings.TrimSpace(os.Args[3]),
		}
		if err = login.Do(); err != nil {
			panic(err)
			return
		}
	} else if strings.TrimSpace(os.Args[1]) == "logout" {
		var willDeleteCookie = false
		if len(os.Args) > 2 {
			willDeleteCookie = strings.TrimSpace(os.Args[2]) == "delete"
		}
		var logout = Logout{
			Client:           client,
			WillDeleteCookie: willDeleteCookie,
		}
		if err = logout.Do(); err != nil {
			panic(err)
			return
		}
	} else if strings.TrimSpace(os.Args[1]) == "download" {
		var download = Download{
			Client:   client,
			IDOrList: strings.TrimSpace(os.Args[2]),
		}
		if err = download.Do(); err != nil {
			panic(err)
			return
		}
	}
	// ----
}

func setHttpClient() (_ *Client, err error) {
	var cookieJar *cookiejar.Jar
	if cookieJar, err = cookiejar.New(&cookiejar.Options{Filename: ".cookie"}); err != nil {
		return nil, err
	}
	return &Client{
		Client:    &http.Client{Jar: cookieJar},
		UserAgent: UserAgent,
	}, err
}

func getResponseBody(resp *http.Response) (string, error) {
	var bodyBytes, err = ioutil.ReadAll(resp.Body)
	return string(bodyBytes), err
}

func checkIsLoggedIn(resp *http.Response, errMsgWhenFailed string) (_ bool, err error) {
	var body string
	
	if resp.StatusCode != http.StatusOK {
		return false, errors.New(errMsgWhenFailed)
	}
	if body, err = getResponseBody(resp); err != nil {
		return false, err
	}
	
	return regexp.MustCompile(`class="user"`).MatchString(body), nil
}
