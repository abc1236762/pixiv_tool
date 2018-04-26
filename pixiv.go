package main

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	
	"gopkg.in/ini.v1"
)

type Doer interface {
	Do() error
}

// A Pixiv save config data and command data about this app.
// Some functions in Pixiv use panic to throw error because
// these should be fixed before release.
type Pixiv struct {
	Config  *Config
	CmdData map[reflect.Type]CmdData
}

// A CmdData save data of commands about this app.
type CmdData struct {
	Cmd     string
	Help    string
	ArgData map[string]ArgData
}

// A ArgData save data of arguments of each command about this app.
type ArgData struct {
	LongCmd    string
	ShortCmd   string
	Type       reflect.Kind
	Help       string
	IsRequired bool
}

// A Config have each function include values from config.ini and default.
type Config struct {
	*Client `cmd:"-"`
	*Login
	*Logout
	*Download
}

// Do initialize contents of Pixiv and run selected function.
func (p *Pixiv) Do() (err error) {
	var doer Doer
	
	// Set default value of command data and check function of command.
	if p.Config != nil || p.CmdData != nil {
		panic("pixiv: struct \"Pixiv\" should be blank inside when Run")
	}
	p.initCmdData()
	p.checkCmdData()
	
	// Get config value to command data.
	p.initConfig()
	if err = p.loadConfig(); err != nil {
		return err
	}
	
	// Parse command and arguments and run selected function.
	if doer, err = p.makeDoer(); err != nil {
		return err
	}
	return doer.Do()
}

// initCmdData initialize contents of Pixiv.CmdData.
func (p *Pixiv) initCmdData() {
	p.CmdData = map[reflect.Type]CmdData{
		reflect.TypeOf(Login{}): {
			Cmd:  "login",
			Help: "Login Pixiv",
			ArgData: map[string]ArgData{
				"Username": {
					LongCmd:    "username",
					ShortCmd:   "u",
					Type:       reflect.String,
					Help:       "the username of your Pixiv account",
					IsRequired: true,
				},
				"Password": {
					LongCmd:    "password",
					ShortCmd:   "p",
					Type:       reflect.String,
					Help:       "the password of your Pixiv account",
					IsRequired: true,
				},
			},
		},
		reflect.TypeOf(Logout{}): {
			Cmd:  "logout",
			Help: "Logout Pixiv",
			ArgData: map[string]ArgData{
				"WillDeleteCookie": {
					LongCmd:    "delete-cookie",
					ShortCmd:   "d",
					Type:       reflect.Bool,
					Help:       "delete the cookie",
					IsRequired: false,
				},
			},
		},
		reflect.TypeOf(Download{}): {
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

// checkCmdData check contents of Pixiv.CmdData and check each function of
// this app matches the requirements of Pixiv.CmdData or not.
func (p *Pixiv) checkCmdData() {
	var panicMsg []string
	
	for i, types := 0, reflect.TypeOf(p.Config).Elem();
			i < types.NumField(); i++ {
		var (
			cmdField     = types.Field(i)
			cmdFieldType = cmdField.Type.Elem()
			cmdData      CmdData
			haveCmdData  bool
		)
		
		// When field is "Client",
		// the tag of "cmd" must be "-" because it is not a command.
		if cmdField.Name == "Client" && cmdField.Tag.Get("cmd") != "-" {
			panicMsg = append(panicMsg, "tag \"cmd\" of \""+
					cmdField.Name+ "\" should be \"-\"")
		}
		if cmdField.Name != "Client" && cmdField.Tag.Get("cmd") == "-" {
			panicMsg = append(panicMsg, "tag \"cmd\" of \""+
					cmdField.Name+ "\" should not be \"-\"")
		}
		if cmdField.Tag.Get("cmd") == "-" {
			continue
		}
		
		// When field is not "Client",
		// it should have a data in "CmdData" because it is a command.
		if cmdData, haveCmdData = p.CmdData[cmdFieldType]; !haveCmdData {
			panicMsg = append(panicMsg, "\""+
					cmdField.Name+ "\" does not have CmdData")
			continue
		}
		
		for j := 0; j < cmdFieldType.NumField(); j++ {
			var (
				argField    = cmdFieldType.Field(j)
				argData     ArgData
				haveArgData bool
			)
			
			// When field is "Client", the tag of "ini"
			// must be "-" because it is not a option or argument.
			if argField.Name == "Client" && argField.Tag.Get("ini") != "-" {
				panicMsg = append(panicMsg, "tag \"ini\" of \""+
						cmdField.Name+ "."+ argField.Name+ "\" should be \"-\"")
			} else if argField.Name != "Client" {
				
				// When field not exist in "ArgData", the tag of "ini"
				// must be ",omitempty" because it is a option.
				if argData, haveArgData =
						cmdData.ArgData[argField.Name]; !haveArgData {
					if argField.Tag.Get("ini") == ",omitempty" {
						continue
					}
					panicMsg = append(panicMsg, "\"" + cmdField.Name+
							"."+ argField.Name+ "\" does not have ArgData")
				}
				
				// When argument is required, the tag of "ini"
				// must be "-" or ",omitempty", "-" means the argument
				// must be gotten from input, ",omitempty" means the
				// argument can be gotten from input and then ini file.
				// When argument is not required, it should have
				// a default value, so the tag of "ini" must be empty.
				if argData.IsRequired && (argField.Tag.Get("ini") !=
						",omitempty" && argField.Tag.Get("ini") != "-") {
					panicMsg = append(panicMsg, "tag \"ini\" of \""+
							cmdField.Name+ "."+ argField.Name+
							"\" should be \",omitempty\" or \"-\"")
				} else if !argData.IsRequired && (argField.Tag.Get("ini") ==
						",omitempty" || argField.Tag.Get("ini") == "-") {
					panicMsg = append(panicMsg, "tag \"ini\" of \""+
							cmdField.Name+ "."+ argField.Name+
							"\" should not be \",omitempty\" or \"-\"")
				}
				
			}
		}
	}
	
	if len(panicMsg) > 0 {
		panic("pixiv: error(s) occurred when checkCmdData:\n\t" +
				strings.Join(panicMsg, ",\n\t"))
	}
}

// initConfig initialize contents of Pixiv.Config with default values.
func (p *Pixiv) initConfig() {
	p.Config = &Config{
		Client: &Client{
			UserAgent: getUserAgent(),
		},
		Login: &Login{
		},
		Logout: &Logout{
			WillDeleteCookie: false,
		},
		Download: &Download{
			Path: "./",
			Naming: Naming{
				SingleFile:   "<artist.name>/(<work.id>) <work.name>",
				MultipleFile: "<work.page>",
				Folder:       "<artist.name>/(<work.id>) <work.name>",
			},
			Metadata: "",
		},
	}
}

// loadConfig load config.ini and set values to Pixiv.Config.
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

// saveConfig get values of Pixiv.Config and save to config.ini.
func (p *Pixiv) saveConfig() (err error) {
	var config = ini.Empty()
	if err = config.ReflectFrom(p.Config); err != nil {
		return err
	}
	return config.SaveTo("config.ini")
}

// makeDoer make a doer that correspond to command and include arguments.
func (p *Pixiv) makeDoer() (doer Doer, err error) {
	if len(os.Args) <= 1 {
		return nil, throw(p, "command is required")
	}
	if doer = p.getCmdDoer(os.Args[1]); doer == nil {
		return nil, throw(p, "command \""+os.Args[1]+"\" not found")
	}
	if err = p.parseArgs(doer); err != nil {
		return nil, err
	}
	return doer, nil
}

// getCmdDoer get the corresponding doer of command.
func (p *Pixiv) getCmdDoer(cmdStr string) (doer Doer) {
	for cmd, data := range p.CmdData {
		if cmdStr == data.Cmd {
			doer = reflect.ValueOf(p.Config).Elem().
				FieldByName(cmd.Name()).Interface().(Doer)
			reflect.ValueOf(doer).Elem().FieldByName("Client").
				Set(reflect.ValueOf(p.Config.Client))
			return doer
		}
	}
	return nil
}

// parseArgs parse arguments of the command and set to doer.
func (p *Pixiv) parseArgs(doer Doer) (err error) {
	// TODO parseArgs
	var (
		addedArgs, errMsgs []string
		isValue            = false
		args               = os.Args[2:]
		cmdStr             = reflect.TypeOf(doer).Elem().Name()
		argData            = p.CmdData[reflect.TypeOf(doer).Elem()].ArgData
		doerVal            = reflect.ValueOf(doer).Elem()
	)
	
	//
Loop:
	for i, argv := range args {
		var (
			isMatched bool
			argName   string
		)
		if isValue {
			isValue = false
			continue
		}
		for name, data := range argData {
			if argv == "-"+data.ShortCmd || argv == "--"+data.LongCmd {
				addedArgs = append(addedArgs, name)
				argName = name
				isMatched = true
			}
		}
		if !isMatched {
			errMsgs = append(errMsgs, "argument \"" + argv+
					"\" not found in command \""+ cmdStr+ "\"")
			continue
		}
		if argData[argName].Type != reflect.Bool {
			isValue = true
		}
		for _, arg := range addedArgs[:len(addedArgs)-1] {
			if arg == argName {
				errMsgs = append(errMsgs, "argument \"" + "-"+
						argData[argName].ShortCmd+ "\" or \""+ "--"+
						argData[argName].LongCmd+ "\" is duplicated")
				continue Loop
			}
		}
		switch argData[argName].Type {
		case reflect.Bool:
			doerVal.FieldByName(argName).SetBool(true)
		case reflect.Int:
			var argvInt int64
			if argvInt, err = strconv.ParseInt(args[i+1], 10, 64); err != nil {
				errMsgs = append(errMsgs, "value of argument \"" + argv+
						"\" require a number, \""+ args[i+1]+ "\" may not")
			} else {
				doerVal.FieldByName(argName).SetInt(argvInt)
			}
		
		case reflect.String:
			doerVal.FieldByName(argName).SetString(args[i+1])
		}
	}
	
	return nil
}
