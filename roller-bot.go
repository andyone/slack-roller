package main

// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"pkg.re/essentialkaos/slacker.v4"

	"pkg.re/essentialkaos/ek.v9/fsutil"
	"pkg.re/essentialkaos/ek.v9/knf"
	"pkg.re/essentialkaos/ek.v9/log"
	"pkg.re/essentialkaos/ek.v9/mathutil"
	"pkg.re/essentialkaos/ek.v9/options"
	"pkg.re/essentialkaos/ek.v9/rand"
	"pkg.re/essentialkaos/ek.v9/strutil"
	"pkg.re/essentialkaos/ek.v9/usage"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP     = "SlackRoller"
	VERSION = "0.2.0"
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
	OPT_CONFIG = "c:config"
	OPT_HELP   = "h:help"
	OPT_VER    = "v:version"
)

const MAX_ROLLS = 20

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_CONFIG: {Value: "roller-bot.conf"},
	OPT_HELP:   {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:    {Type: options.BOOL, Alias: "ver"},
}

var wrongAttempts = 0
var wrongMessages = []string{
	"_Бро, ты тупой?_",
	"_Ну вот для кого я выше писал, как мной пользоваться?_",
	"_Забей короче, ничего не скажу._",
	"_Еще варианты? Не, реально попробуй, я подожду._",
	"_Я состою из 200 строк кода и то умнее тебя._",
	"_Давай по душам, ты в школе учился?_",
}

// ////////////////////////////////////////////////////////////////////////////////// //

func main() {
	_, errs := options.Parse(optMap)

	if len(errs) != 0 {
		fmt.Println("Error while arguments parsing:")

		for _, err := range errs {
			fmt.Printf("  %v\n", err)
		}

		os.Exit(1)
	}

	if options.GetB(OPT_HELP) {
		showUsage()
		return
	}

	if options.GetB(OPT_VER) {
		showAbout()
		return
	}

	loadConfig()
	setupLog()

	startBot()
}

func loadConfig() {
	var err error
	var confPath = options.GetS(OPT_CONFIG)

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
	bot.ErrorHandler = errorHandler
	bot.UnknownCommandHandler = unknownCommandHandler

	bot.CommandHandlers = map[string]slacker.CommandHandler{
		"roll":    rollCommandHandler,
		"брось":   rollCommandHandler,
		"бросить": rollCommandHandler,
		"кинь":    rollCommandHandler,
		"random":  sampleCommandHandler,
		"sample":  sampleCommandHandler,
		"выбери":  sampleCommandHandler,
		"help":    helpCommandHandler,
		"usage":   helpCommandHandler,
		"помощь":  helpCommandHandler,
		"помоги":  helpCommandHandler,
	}

	err := bot.Run()

	if err != nil {
		log.Crit(err.Error())
		os.Exit(1)
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

func errorHandler(err error) {
	log.Error("Got error from Slack API: %v", err)
}

func connectHandler() {
	log.Info("Bot successfully connected to Slack!")
}

func helloHandler() string {
	log.Debug("Executed hello handler")
	return "Всем чмоке в этом чате! :kissing_closed_eyes:"
}

func unknownCommandHandler(user slacker.User, cmd string, args []string) string {
	log.Debug("Got unknown command: %s → %s %s", user.RealName, cmd, strings.Join(args, " "))

	wrongAttempts++

	if wrongAttempts > 3 {
		return wrongMessages[rand.Int(len(wrongMessages)-1)]
	}

	return "Ничего не понял. Если не знаешь как мной пользоваться, просто напиши _help_."
}

func rollCommandHandler(user slacker.User, args []string) []string {
	log.Debug("Got roll command: %s → %s", user.RealName, strings.Join(args, " "))

	wrongAttempts = 0

	args = append(args, "", "")

	return []string{rollDice(args[0], args[1])}
}

func sampleCommandHandler(user slacker.User, args []string) []string {
	log.Debug("Got sample command: %s → %s", user.RealName, strings.Join(args, " "))

	wrongAttempts = 0

	return []string{sample(args)}
}

func helpCommandHandler(user slacker.User, args []string) []string {
	log.Debug("Got help command: %s → %s", user.RealName, strings.Join(args, " "))

	wrongAttempts = 0

	return []string{getHelpContent()}
}

// ////////////////////////////////////////////////////////////////////////////////// //

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

func sample(args []string) string {
	if len(args) <= 1 {
		return "Нужно больше вариантов для выбора."
	}

	samples := strutil.Fields(strings.Join(args, " "))
	samplesCount := len(samples)
	selectedItem := rand.Int(samplesCount)

	return fmt.Sprintf("Я выбрал *%s*.", samples[selectedItem])
}

func getHelpContent() string {
	var content string

	content += "*Команды:*\n"
	content += "`roll` - Бросить кубик\n"
	content += "`roll sides` - Бросить кубик с указанным количеством сторон\n"
	content += "`roll sides count` - Бросить кубик с указанным количеством сторон, указанное количество раз\n"
	content += "`sample samples...` - Выбрать один из данных вариантов\n"
	content += "\n"
	content += "*Примеры:*\n"
	content += "`roll` _Просто бросить кубик_\n"
	content += "`roll 12` _Бросить кубик с 12 сторонами_\n"
	content += "`roll 12 5` _Бросить кубик с 12 сторонами 5 раз_\n"
	content += "`sample Вася Петя Нина` _Выбрать один из трех вариантов_\n"
	content += "`sample \"Вася П.\" \"Петя Г.\" \"Нина З.\"` _Выбрать один из трех вариантов состоящих из нескольких слов_\n"
	content += "\n"

	return content
}

// ////////////////////////////////////////////////////////////////////////////////// //

func showUsage() {
	info := usage.NewInfo("")

	info.AddOption(OPT_CONFIG, "Path to config file", "file")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

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
