package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
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
	CookieFileName = ".cookie"
)

// An AppError is a implementation of error interface for this app.
type AppError struct {
	Prefix string
	Msg    string
}

// Error is needed when implement an error interface.
func (ae *AppError) Error() string {
	return ae.Prefix + ": " + ae.Msg
}

// throw return an error interface made by AppError.
func throw(doer Doer, msg string) error {
	return &AppError{
		Prefix: strings.ToLower(reflect.TypeOf(doer).Elem().Name()),
		Msg:    msg,
	}
}

// setHttpClient set a Client with a cookieJar and user agent.
func setHttpClient() (_ *Client, err error) {
	var cookieJar *cookiejar.Jar
	if cookieJar, err = cookiejar.New(
		&cookiejar.Options{Filename: CookieFileName}); err != nil {
		return nil, err
	}
	return &Client{
		Client:    &http.Client{Jar: cookieJar},
		UserAgent: UserAgent,
	}, err
}

// getResponseBody get string of body from http response.
func getResponseBody(resp *http.Response) (string, error) {
	var bodyBytes, err = ioutil.ReadAll(resp.Body)
	return string(bodyBytes), err
}

// checkIsLoggedIn check that this app is logged in on Pixiv or not.
func checkIsLoggedIn(resp *http.Response, doer Doer, failedMsg string) (_ bool, err error) {
	var body string
	if resp.StatusCode != http.StatusOK {
		return false, throw(doer, failedMsg)
	}
	if body, err = getResponseBody(resp); err != nil {
		return false, err
	}
	return regexp.MustCompile(`class="user"`).MatchString(body), nil
}

func main() {
	var (
		client *Client
		err    error
	)
	
	// test
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			os.Exit(1)
		}
	}()
	
	if len(os.Args) == 1 {
		new(Pixiv).Init()
		return
	}
	
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
