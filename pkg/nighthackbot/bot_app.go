package nighthackbot

import (
	"context"
	"fmt"
	"html"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type BotApp struct {
	Config  *Config
	Bot     *tgbotapi.BotAPI
	BotName string
	DB      *gorm.DB

	// services
	AskService   *AskService
	UsersService *UsersService

	// commands
	Commands []Command
}

func NewBotApp() (a *BotApp) {
	a = &BotApp{
		Config: &Config{},
	}
	a.AskService = NewAskService(a)
	a.UsersService = NewUsersService(a)
	a.Commands = []Command{
		&AdminCommand{App: a},
		&StartCommand{App: a},
	}
	return
}

func (app *BotApp) Run() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Str("version", Version).Msgf("Starting nighthack-bot")

	// load config
	if err := app.LoadConfig(); err != nil {
		log.Fatal().Msgf("Failed to load config: %s", err)
	}

	// init db
	if err := app.InitDB(); err != nil {
		log.Fatal().Msgf("Failed to init db: %s", err)
	}

	// init telegram
	if err := app.InitTelegram(); err != nil {
		log.Fatal().Msgf("Failed to init telegram: %s", err)
	}

	// run loop
	if err := app.RunLoop(); err != nil {
		log.Fatal().Msgf("Failed to run loop: %s", err)
	}
}

func (app *BotApp) LoadConfig() error {
	viper.SetConfigName("nighthackbot-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/nighthackbot/")
	viper.AddConfigPath("$HOME/.config/alufers/nighthackbot")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("error reading config file: %s", err)
	}
	err = viper.Unmarshal(app.Config)
	if err != nil {
		return fmt.Errorf("error unmarshalling config: %s", err)
	}
	log.Info().Str("from", viper.ConfigFileUsed()).Msgf("Loaded config")
	return nil
}

func (app *BotApp) InitTelegram() error {
	bot, err := tgbotapi.NewBotAPI(app.Config.Telegram.Token)
	if err != nil {
		return fmt.Errorf("error creating bot: %s", err)
	}
	bot.Debug = app.Config.Telegram.Debug
	app.Bot = bot
	me, err := app.Bot.GetMe()
	if err != nil {
		return fmt.Errorf("error getting bot info: %s", err)
	}

	log.Info().Str("bot_name", me.UserName).Bool("CanReadAllGroupMessages", me.CanReadAllGroupMessages).Msgf("Bot started")
	app.BotName = me.UserName
	return nil
}

func (app *BotApp) InitDB() error {
	var db *gorm.DB
	var err error
	if app.Config.DB.Type == "" {
		return fmt.Errorf("db type is not set in config")
	}
	switch app.Config.DB.Type {
	case "postgres":
		db, err = gorm.Open(postgres.Open(app.Config.DB.DSN), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(app.Config.DB.Filename), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	default:
		return fmt.Errorf("unknown db type: %s", app.Config.DB.Type)
	}
	if err != nil {
		return fmt.Errorf("error opening db: %s", err)
	}
	app.DB = db

	if err := app.DB.AutoMigrate(&User{}); err != nil {
		return fmt.Errorf("error auto-migrating db: %s", err)
	}

	log.Info().Str("db_type", app.Config.DB.Type).Msgf("DB initialized")

	return nil
}

func (app *BotApp) RunLoop() error {
	myCommands := []tgbotapi.BotCommand{}
	for _, cmd := range app.Commands {
		rawCmd := strings.TrimPrefix(strings.Split(cmd.Aliases()[0], " ")[0], "/")
		myCommands = append(myCommands, tgbotapi.BotCommand{
			Command:     rawCmd,
			Description: cmd.Help(),
		})
	}

	commandsConfig := tgbotapi.NewSetMyCommands(myCommands...)

	if _, err := app.Bot.Request(commandsConfig); err != nil {
		return fmt.Errorf("failed to set my commands: %v", err)
	}
	log.Info().Msgf("Receiving messages...")
	u := tgbotapi.NewUpdate(0)
	u.AllowedUpdates = []string{"message", "inline_query", "callback_query", "edited_message"}
	u.Timeout = 60

	updates := app.Bot.GetUpdatesChan(u)
	for u := range updates {
		go func(update tgbotapi.Update) {
			log.Printf("incoming message: %+v", update)
			if app.AskService.ProcessIncomingMessage(update) {
				return
			}
			var err error
			var cmdText string

			args := &CommandArguments{
				BotApp:         app,
				update:         &update,
				namedArguments: map[string]string{},
			}
			if update.Message != nil {
				cmdText = update.Message.Text
				args.ChatID = update.Message.Chat.ID
				args.FromUserID = update.Message.From.ID
				args.FromUserName = update.Message.From.UserName
			}
			if update.CallbackQuery != nil {
				cmdText = update.CallbackQuery.Data
				args.ChatID = update.CallbackQuery.Message.Chat.ID
				args.FromUserID = update.CallbackQuery.From.ID
				args.FromUserName = update.CallbackQuery.From.UserName
			}
			seg := strings.Split(cmdText, " ")
			args.CommandName = seg[0]
			args.Arguments = seg[1:]
			didFind := false
			for _, cmd := range app.Commands {

				if CommandMatches(app, cmd, cmdText) {
					args.Command = cmd
					for i, argTpl := range cmd.Arguments() {
						if argTpl.Variadic {
							args.namedArguments[argTpl.Name] = strings.Join(args.Arguments[i:], " ")
							break
						}
						if i >= len(args.Arguments) {
							break
						}
						args.namedArguments[argTpl.Name] = args.Arguments[i]
					}
					if usersError := app.UsersService.AddUserToArgs(args); usersError != nil {
						err = usersError
						break
					}
					ctx := context.TODO()
					err = cmd.Execute(ctx, args)
					didFind = true
					break
				}
			}
			if !didFind && update.CallbackQuery != nil {
				app.Bot.Send(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			}
			if err != nil {
				log.Printf("Error while processing command %v: %v", cmdText, err)
				if args.update.CallbackQuery != nil {
					msg := tgbotapi.NewCallback(args.update.CallbackQuery.ID, "🚫 Error:"+err.Error())
					app.Bot.Send(msg)
				} else {

					msg := tgbotapi.NewMessage(args.ChatID, "🚫 Error: <b>"+html.EscapeString(err.Error())+"</b>")
					msg.ParseMode = "HTML"
					if update.Message != nil {
						msg.ReplyToMessageID = update.Message.MessageID
					}
					app.Bot.Send(msg)
				}
			}
		}(u)
	}
	return nil
}
