package main

import (
	"net/http"
	"os"
	
	"github.com/juju/persistent-cookiejar"
)

type Logout struct {
	Client           *Client `ini:"-"`
	WillDeleteCookie bool
}

func (l *Logout) Do() (err error) {
	if err = l.logout(); err != nil {
		return err
	}
	
	// Delete or update cookie file
	if l.WillDeleteCookie {
		if err = os.Remove(".cookie"); err != nil {
			return err
		}
	} else {
		l.Client.Jar.(*cookiejar.Jar).Save()
	}
	
	return nil
}

func (l *Logout) logout() (err error) {
	var (
		resp       *http.Response
		isLoggedIn bool
	)
	
	// Check that pixiv is already logged or not
	if resp, err = l.Client.Get(PixivHomeURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if isLoggedIn, err = checkIsLoggedIn(resp, l,
		"request status is not OK when checking that not login yet or not"); err != nil {
		return err
	} else if !isLoggedIn {
		return throw(l, "not logged in yet")
	}
	
	// Send a GET request to logout
	if resp, err = l.Client.Get(PixivLogoutURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return throw(l, "request status is not OK when logging out")
	}
	
	// Check that it logged out successful or not
	if resp, err = l.Client.Get(PixivHomeURL); err != nil {
		return err
	}
	defer resp.Body.Close()
	if isLoggedIn, err = checkIsLoggedIn(resp, l,
		"request status is not OK when checking that logout successful or not"); err != nil {
		return err
	} else if isLoggedIn {
		return throw(l, "logout failed")
	}
	
	return nil
}
