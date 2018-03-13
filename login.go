package main

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	
	"github.com/juju/persistent-cookiejar"
)

type Login struct {
	Client   *Client `ini:"-"`
	Username string  `ini:",omitempty"`
	Password string  `ini:"-"`
}

func (l *Login) Do() (err error) {
	if err = l.login(); err != nil {
		return err
	}
	
	l.Client.Jar.(*cookiejar.Jar).Save()
	
	return nil
}

func (l *Login) login() (err error) {
	var (
		resp       *http.Response
		postKey    string
		isLoggedIn bool
	)
	
	// Check that pixiv is already logged or not
	if resp, err = l.Client.Get(PixivHomeURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if isLoggedIn, err = checkIsLoggedIn(resp,
		"login: request status is not OK when checking that login already or not"); err != nil {
		return err
	} else if isLoggedIn {
		return errors.New("login: already logged in")
	}
	
	// Get post key and send a POST request to login
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
		return errors.New("login: request status is not OK when logging in")
	}
	
	// Check that it logged in successful or not
	if isLoggedIn, err = checkIsLoggedIn(resp,
		"login: request status is not OK when checking that login successful or not"); err != nil {
		return err
	} else if !isLoggedIn {
		return errors.New("login: login failed, please check username and password")
	}
	
	return nil
}

func (l *Login) getPostKey() (_ string, err error) {
	var (
		resp *http.Response
		body string
	)
	
	// Send a GET request to get the "post_key"
	if resp, err = l.Client.Get(PixivLoginURL); err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("login: request status is not OK when getting post key")
	}
	if body, err = getResponseBody(resp); err != nil {
		return "", err
	}
	
	return regexp.MustCompile(
		`<input.*?name="post_key".*?value="(.*?)".*?>`).
		FindStringSubmatch(body)[1], err
}
