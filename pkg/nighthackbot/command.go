package nighthackbot

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandDefArgument struct {
	Name        string
	Description string
	Question    string
	Variadic    bool
}

type Command interface {
	Helpable
	Aliases() []string
	Arguments() []*CommandDefArgument
	Execute(ctx context.Context, args *CommandArguments) error
}

type Helpable interface {
	Help() string
}

type CommandArguments struct {
	BotApp         *BotApp
	update         *tgbotapi.Update
	CommandName    string
	Arguments      []string
	ChatID         int64
	FromUserID     int64
	FromUserName   string
	namedArguments map[string]string
	Command        Command
	User           *User
}

func (a *CommandArguments) GetOrAskForArgument(name string, suggestionsArr ...map[string]string) (string, error) {
	if val, ok := a.namedArguments[name]; ok {
		return val, nil
	}
	var cmdTemplate *CommandDefArgument
	for _, arg := range a.Command.Arguments() {
		if arg.Name == name {
			cmdTemplate = arg
			break
		}
	}
	if cmdTemplate == nil {
		return "", nil
	}
	return a.BotApp.AskService.AskForArgument(a.ChatID, "❓ "+cmdTemplate.Question, suggestionsArr...)
}

func CommandMatches(BotApp *BotApp, cmd Command, userInput string) bool {
	usersCmd := strings.Split(userInput, " ")[0]
	// strip bot suffix on groups
	usersCmd = strings.TrimSuffix(usersCmd, "@"+BotApp.BotName)
	for _, alias := range cmd.Aliases() {
		if alias == usersCmd {
			return true
		}
	}
	return false
}
