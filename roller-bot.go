package main

// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"pkg.re/essentialkaos/slacker.v1"

	"pkg.re/essentialkaos/ek.v1/arg"
	"pkg.re/essentialkaos/ek.v1/fsutil"
	"pkg.re/essentialkaos/ek.v1/knf"
	"pkg.re/essentialkaos/ek.v1/log"
	"pkg.re/essentialkaos/ek.v1/mathutil"
	"pkg.re/essentialkaos/ek.v1/rand"
	"pkg.re/essentialkaos/ek.v1/usage"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP     = "SlackRoller"
	VERSION = "0.0.3"
)

const (
	BOT_NAME  = "bot:name"
	BOT_TOKEN = "bot:token"
	LOG_DIR   = "log:dir"
	LOG_FILE  = "log:file"
	LOG_PERMS = "log:perms"
	LOG_LEVEL = "log:level"
)

const (
	ARG_CONFIG = "c:config"
	ARG_HELP   = "h:help"
	ARG_VER    = "v:version"
)

const MAX_ROLLS = 20

// ////////////////////////////////////////////////////////////////////////////////// //

var argMap = arg.Map{
	ARG_CONFIG: &arg.V{Value: "roller-bot.conf"},
	ARG_HELP:   &arg.V{Type: arg.BOOL, Alias: "u:usage"},
	ARG_VER:    &arg.V{Type: arg.BOOL, Alias: "ver"},
}

var wrongAttempts = 0
var wrongMessages = []string{
	"_Ты тупой?_",
	"_Ну вот для кого я выше писал, как мной пользоваться?_",
	"_Забей короче, ничего не скажу._",
	"_Еще варианты? Не, реально попробуй, я подожду._",
	"_Я состою из 200 строк кода и то умнее тебя._",
	"_Давай по душам, ты в школе учился?_",
}

// ////////////////////////////////////////////////////////////////////////////////// //

func main() {
	_, errs := arg.Parse(argMap)

	if len(errs) != 0 {
		fmt.Println("Error while arguments parsing:")

		for _, err := range errs {
			fmt.Printf("  %v\n", err)
		}

		os.Exit(1)
	}

	if arg.GetB(ARG_HELP) {
		showUsage()
		return
	}

	if arg.GetB(ARG_VER) {
		showAbout()
		return
	}

	loadConfig()
	setupLog()

	startBot()
}

func loadConfig() {
	var err error
	var confPath = arg.GetS(ARG_CONFIG)

	switch {
	case !fsutil.IsExist(confPath):
		fmt.Printf("Config %s is not exist\n", confPath)
		os.Exit(1)

	case !fsutil.IsReadable(confPath):
		fmt.Printf("Config %s is not readable\n", confPath)
		os.Exit(1)

	case !fsutil.IsNonEmpty(confPath):
		fmt.Printf("Config %s is empty\n", confPath)
		os.Exit(1)
	}

	err = knf.Global(confPath)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func setupLog() {
	err := log.Set(knf.GetS(LOG_FILE), knf.GetM(LOG_PERMS))

	if err != nil {
		fmt.Printf("Error with log setup: %v\n", err)
		os.Exit(1)
	}

	err = log.MinLevel(knf.GetS(LOG_LEVEL))

	if err != nil {
		fmt.Printf("Error with changing log level: %v\n", err)
	}
}

func startBot() {
	bot := slacker.NewBot(knf.GetS(BOT_NAME), knf.GetS(BOT_TOKEN))

	// Setup async handlers
	bot.ConnectHandler = connectHandler
	bot.HelloHandler = helloHandler
	bot.CommandHandler = commandHandler
	bot.ErrorHandler = errorHandler

	err := bot.Run()

	if err != nil {
		log.Crit(err.Error())
		os.Exit(1)
	}
}

func errorHandler(err error) {
	log.Error("Got error from Slack API: %v", err)
}

func connectHandler() {
	log.Info("Bot successfully connected to Slack!")
}

func helloHandler() string {
	log.Debug("Executed hello handler")
	return "Всем чмоке в этом чате!"
}

func commandHandler(command string, args []string) string {
	log.Debug("Got command: %s %s", command, strings.Join(args, " "))

	switch command {
	case "roll", "брось", "бросить", "кинь":
		wrongAttempts = 0
		args = append(args, "", "")
		return []string{rollDice(args[0], args[1])}

	default:
		wrongAttempts++

		if wrongAttempts > 3 {
			return []string{wrongMessages[rand.Int(len(wrongMessages)-1)]}
		}

		return []string{"Ничего не понял, напиши roll и я кину кубик."}
	}
}

func rollDice(sides, count string) string {
	sidesInt := 5
	countInt := 1

	if sides != "" {
		si, err := strconv.Atoi(sides)

		if err == nil {
			sidesInt = si
		}
	}

	if count != "" {
		ci, err := strconv.Atoi(count)

		if err == nil {
			countInt = ci
		}
	}

	sidesInt = mathutil.Between(sidesInt, 2, 10000000)
	countInt = mathutil.Between(countInt, 1, MAX_ROLLS)

	result := []string{}

	for i := 0; i < countInt; i++ {
		result = append(result, fmt.Sprintf("*%d*", rand.Int(sidesInt)+1))
	}

	return fmt.Sprintf("Кубик брошен. Выпало %s.", strings.Join(result, ", "))
}

// ////////////////////////////////////////////////////////////////////////////////// //

func showUsage() {
	info := usage.NewInfo("")

	info.AddOption(ARG_CONFIG, "Path to config file", "file")
	info.AddOption(ARG_HELP, "Show this help message")
	info.AddOption(ARG_VER, "Show version")

	info.Render()
}

func showAbout() {
	about := &usage.About{
		App:     APP,
		Version: VERSION,
		Desc:    "Slack bot for rolling dice",
		License: "MIT",
	}

	about.Render()
}
