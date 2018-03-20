package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	
	"github.com/juju/persistent-cookiejar"
)

const (
	UserAgentFmt   = "Mozilla/5.0 (%s rv:%d.0) Gecko/%s Firefox/%d.0"
	PixivHomeURL   = "https://www.pixiv.net/"
	PixivLoginURL  = "https://accounts.pixiv.net/login?lang=ja&source=pc&view_type=page&ref=wwwtop_accounts_index"
	PixivLogoutURL = PixivHomeURL + "logout.php?return_to=%2F"
	PixivWorkURL   = PixivHomeURL + "member_illust.php?mode=medium&illust_id=%s"
	PixivMangaURL  = PixivHomeURL + "member_illust.php?mode=manga_big&illust_id=%s&page=%d"
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

// getUserAgent get default user agent of this app in each os.
func getUserAgent() string {
	// Value of browserVer and userAgentOS must update regularly.
	var browserVer uint32 = 59
	var userAgentOS, geckoVer = func() (string, string) {
		switch runtime.GOOS {
		case "windows":
			return "Windows NT 10.0; Win64; x64;", "20100101"
		case "darwin":
			return "Macintosh; Intel Mac OS X 10.13;", "20100101"
		case "android":
			return "Android 8.1.0; Tablet;",
					fmt.Sprintf("%d.0", browserVer)
		default:
			return "X11; Linux x86_64;", "20100101"
		}
	}()
	return fmt.Sprintf(UserAgentFmt,
		userAgentOS, browserVer, geckoVer, browserVer)
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

// setHttpClient set a Client with a cookieJar and user agent.
// TODO: temporarily function, remove after complete function of process command.
func setHttpClient() (_ *Client, err error) {
	var cookieJar *cookiejar.Jar
	if cookieJar, err = cookiejar.New(
		&cookiejar.Options{Filename: CookieFileName}); err != nil {
		return nil, err
	}
	return &Client{
		Client:    &http.Client{Jar: cookieJar},
		UserAgent: getUserAgent(),
	}, err
}

func main() {
	var (
		client *Client
		err    error
	)
	
	// TODO: temporarily test, remove after complete function of process command.
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			os.Exit(1)
		}
	}()
	
	if len(os.Args) == 1 {
		var pixiv = Pixiv{}
		
		pixiv.Init()
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
	// End test.
	
}
