package main

import (
	"net/http"
	"net/url"
	"regexp"
	
	"github.com/juju/persistent-cookiejar"
)

// A Login process login in this app.
type Login struct {
	Client   *Client `ini:"-"`
	Username string  `ini:",omitempty"`
	Password string  `ini:"-"`
}

// Do run login process in this app.
func (l *Login) Do() (err error) {
	if err = l.login(); err != nil {
		return err
	}
	// After logged in, save cookieJar.
	l.Client.Jar.(*cookiejar.Jar).Save()
	return nil
}

// login make this app log in to Pixiv.
func (l *Login) login() (err error) {
	var (
		resp       *http.Response
		postKey    string
		isLoggedIn bool
	)
	
	// Check that Pixiv is already logged or not.
	if resp, err = l.Client.Get(PixivHomeURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if isLoggedIn, err = checkIsLoggedIn(resp, l,
		"request status is not OK when checking " +
			"that login already or not"); err != nil {
		return err
	} else if isLoggedIn {
		return throw(l, "already logged in")
	}
	
	// Get post key and send a POST request to login.
	if postKey, err = l.getPostKey(); err != nil {
		return err
	}
	if resp, err = l.Client.PostForm(PixivLoginURL, url.Values{
		"pixiv_id":  []string{l.Username},
		"password":  []string{l.Password},
		"post_key":  []string{postKey},
		"source":    []string{"pc"},
		"return_to": []string{PixivHomeURL},
		"ref":       []string{"wwwtop_accounts_index"},
	}); err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return throw(l, "request status is not OK when logging in")
	}
	
	// Check that it logged in successful or not.
	if isLoggedIn, err = checkIsLoggedIn(resp, l,
		"request status is not OK when checking " +
			"that login successful or not"); err != nil {
		return err
	} else if !isLoggedIn {
		return throw(l, "login failed, please check username and password")
	}
	
	return nil
}

// getPostKey get "post_key" that is needed when login Pixiv.
func (l *Login) getPostKey() (_ string, err error) {
	var (
		resp *http.Response
		body string
	)
	
	// Send a GET request to get the "post_key".
	if resp, err = l.Client.Get(PixivLoginURL); err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", throw(l, "request status is not OK when getting post key")
	}
	if body, err = getResponseBody(resp); err != nil {
		return "", err
	}
	
	return regexp.MustCompile(
		`<input.*?name="post_key".*?value="(.*?)".*?>`).
			FindStringSubmatch(body)[1], err
}
