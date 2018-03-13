package main

import (
	"fmt"
	"os"
	"reflect"
	
	"gopkg.in/ini.v1"
)

type Doer interface {
	Do() error
}

type Pixiv struct {
	Config  *Config
	CmdData map[string]CmdData
}

type CmdData struct {
	Cmd     string
	Help    string
	ArgData map[string]ArgData
}

type ArgData struct {
	LongCmd      string
	ShortCmd     string
	Type         reflect.Kind
	Help         string
	IsRequired   bool
}

type Config struct {
	*Client `cmd:"-"`
	*Login
	*Logout
	*Download
}

func (p *Pixiv) Init() {
	if p.Config != nil || p.CmdData != nil {
		panic("pixiv.Init: struct \"Pixiv\" should be blank inside")
	}
	p.initCmdData()
	p.checkCmdData()
	p.initConfig()
	p.loadConfig()
}

func (p *Pixiv) initCmdData() {
	p.CmdData = map[string]CmdData{
		"Login": {
			Cmd:  "login",
			Help: "Login Pixiv",
			ArgData: map[string]ArgData{
				"Username": {
					LongCmd:    "username",
					ShortCmd:   "u",
					Type:       reflect.String,
					Help:       "The username of your Pixiv account",
					IsRequired: true,
				},
				"Password": {
					LongCmd:    "password",
					ShortCmd:   "p",
					Type:       reflect.String,
					Help:       "The password of your Pixiv account",
					IsRequired: true,
				},
			},
		},
		"Logout": {
			Cmd:  "logout",
			Help: "Logout Pixiv",
			ArgData: map[string]ArgData{
				"WillDeleteCookie": {
					LongCmd:    "delete-cookie",
					ShortCmd:   "d",
					Type:       reflect.Bool,
					Help:       "Delete the cookie",
					IsRequired: false,
				},
			},
		},
		"Download": {
			Cmd:  "download",
			Help: "Download a work from the ID or works from a list in Pixiv",
			ArgData: map[string]ArgData{
				"IDOrList": {
					LongCmd:    "id-or-list",
					ShortCmd:   "i",
					Type:       reflect.String,
					Help:       "the work ID that want to download in Pixiv or filename of the list that generate by other commands",
					IsRequired: true,
				},
				"Path": {
					LongCmd:    "path",
					ShortCmd:   "p",
					Type:       reflect.String,
					Help:       "where the download file(s) will be save, must be a folder",
					IsRequired: false,
				},
			},
		},
	}
}

func (p *Pixiv) checkCmdData() {
	var types = reflect.TypeOf(p.Config).Elem()
	
	for i := 0; i < types.NumField(); i++ {
		var (
			cmdField     = types.Field(i)
			cmdFieldType = cmdField.Type.Elem()
			cmdData      CmdData
			haveCmdData  bool
		)
		
		// When field is "Client", the tag of "cmd" must be "-" because it is not a command
		if cmdField.Name == "Client" && cmdField.Tag.Get("cmd") != "-" {
			panic(fmt.Sprintf("pixiv.checkCmdData: tag \"cmd\" of \"%s\" should be \"-\"", cmdField.Name))
		} else if cmdField.Name != "Client" && cmdField.Tag.Get("cmd") == "-" {
			panic(fmt.Sprintf("pixiv.checkCmdData: tag \"cmd\" of \"%s\" should not be \"-\"", cmdField.Name))
		}
		if cmdField.Tag.Get("cmd") == "-" {
			continue
		}
		
		// When field is not "Client", it should have a data in "CmdData" because it is a command
		if cmdData, haveCmdData = p.CmdData[cmdField.Name]; !haveCmdData {
			panic(fmt.Sprintf("pixiv.checkCmdData: %s does not have CmdData", cmdField.Name))
		} else {
			for j := 0; j < cmdFieldType.NumField(); j++ {
				var (
					argField    = cmdFieldType.Field(j)
					argData     ArgData
					haveArgData bool
				)
				
				// When field is "Client", the tag of "ini" must be "-" because it is not a option or argument
				if argField.Name == "Client" && argField.Tag.Get("ini") != "-" {
					panic(fmt.Sprintf("pixiv.checkCmdData: tag \"ini\" of \"%s.%s\" should be \"-\"",
						cmdField.Name, argField.Name))
				} else if argField.Name != "Client" {
					// When field not exist in "ArgData", the tag of "ini" must be ",omitempty" because it is a option
					if argData, haveArgData = cmdData.ArgData[argField.Name]; !haveArgData {
						if argField.Tag.Get("ini") == ",omitempty" {
							continue
						}
						panic(fmt.Sprintf("pixiv.checkCmdData: \"%s.%s\" does not have ArgData",
							cmdField.Name, argField.Name))
					}
					// When argument is required, the tag of "ini" must be "-" or ",omitempty" and
					// when "-", the argument must be gotten from input, when ",omitempty", the argument can be gotten
					// from input and then ini file.
					// When argument is not required, it should have a default value, so the tag of "ini" must be empty
					if argData.IsRequired && (argField.Tag.Get("ini") != ",omitempty" && argField.Tag.Get("ini") != "-") {
						panic(fmt.Sprintf("pixiv.checkCmdData: tag \"ini\" of \"%s.%s\" should be \",omitempty\" or \"-\"",
							cmdField.Name, argField.Name))
					} else if !argData.IsRequired && (argField.Tag.Get("ini") == ",omitempty" || argField.Tag.Get("ini") == "-") {
						panic(fmt.Sprintf("pixiv.checkCmdData: tag \"ini\" of \"%s.%s\" should not be \",omitempty\" or \"-\"",
							cmdField.Name, argField.Name))
					}
				}
			}
		}
	}
}

func (p *Pixiv) initConfig() {
	p.Config = &Config{
		Client: &Client{
			UserAgent: UserAgent,
		},
		Login: &Login{
		},
		Logout: &Logout{
			WillDeleteCookie: false,
		},
		Download: &Download{
			Path: "./",
			Naming: Naming{
				SingleFile:   "<artist.name>/(<work_id>) <work_name>",
				MultipleFile: "<work_page>",
				Folder:       "<artist_name>/(<work_id>) <work_name>",
			},
			Metadata: "",
		},
	}
}

func (p *Pixiv) loadConfig() (err error) {
	var config *ini.File
	
	if _, err = os.Stat("config.ini"); os.IsNotExist(err) {
		if err = p.saveConfig(); err != nil {
			return err
		}
	}
	if config, err = ini.Load("config.ini"); err != nil {
		return err
	}
	
	return config.MapTo(p.Config)
}

func (p *Pixiv) saveConfig() (err error) {
	var config = ini.Empty()
	
	if err = config.ReflectFrom(p.Config); err != nil {
		return err
	}
	
	return config.SaveTo("config.ini")
}
