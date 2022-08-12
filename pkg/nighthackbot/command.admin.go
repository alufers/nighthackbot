package nighthackbot

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

type AdminCommand struct {
	App *BotApp
}

func (s *AdminCommand) Aliases() []string {
	return []string{"/admin"}
}

func (s *AdminCommand) Arguments() []*CommandDefArgument {
	return []*CommandDefArgument{{
		Name: "command",
	}}
}

func (f *AdminCommand) Help() string {
	return "shows a menu for admin commands"
}

func (f *AdminCommand) Execute(ctx context.Context, args *CommandArguments) error {
	subcommands := map[string]func(ctx context.Context, args *CommandArguments) error{
		"add_admin_user":                f.addAdminUser,
		"remove_admin_user":             f.removeAdminUser,
		"set_call_for_volounteers_time": nil,
		"set_nighthack_time":            nil,
		"force_next_nighthack":          nil,
		"cancel_next_nighthack":         nil,
		"override_next_nighthack_time":  nil,
	}
	if args.namedArguments["command"] == "" {
		admins := []User{}
		if err := f.App.DB.Where("is_admin = ?", true).Find(&admins).Error; err != nil {
			return err
		}
		adminsStr := ""
		for _, admin := range admins {
			if adminsStr != "" {
				adminsStr += ", "
			}
			if admin.Username != "" {
				adminsStr += "@" + admin.Username
			} else {
				adminsStr += strconv.FormatInt(admin.TelegramID, 10)
			}
		}
		msg := tgbotapi.NewMessage(args.update.Message.Chat.ID, fmt.Sprintf("Current admins: %v\n\nAdmin options:", adminsStr))
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("--- 🔧 General settings ---", "null"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👤 Add admin user", "/admin add_admin_user"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Remove admin user", "/admin remove_admin_user"),
			),

			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➡️⏰ Set call for volounteers time", "/admin set_call_for_volounteers_time"),
				tgbotapi.NewInlineKeyboardButtonData("➡️🕑 Set nighthack time", "/admin set_nighthack_time"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("--- 🎟️ Next nighthack ---", "null"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💪 Force next nighthack", "/admin force_next_nighthack"),
				tgbotapi.NewInlineKeyboardButtonData("🚫 Cancel next nighthack", "/admin cancel_next_nighthack"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🕑 Override next nighthack time", "/admin override_next_nighthack_time"),
			),
		)
		_, err := f.App.Bot.Send(
			msg,
		)
		return err
	}

	if subcommand, ok := subcommands[args.namedArguments["command"]]; ok && subcommand != nil {
		if args.update.CallbackQuery != nil {
			f.App.Bot.Send(tgbotapi.NewCallback(args.update.CallbackQuery.ID, ""))
			args.update.CallbackQuery = nil
		}
		return subcommand(ctx, args)
	} else {
		return fmt.Errorf("unknown admin subcommand %q", args.namedArguments["command"])
	}
}

func (f *AdminCommand) addAdminUser(ctx context.Context, args *CommandArguments) error {
	result, err := f.App.AskService.AskForArgument(args.ChatID, "Enter telegram <b>USER ID</b> for the new admin:\nTip: you can use https://t.me/username_to_id_bot")
	if err != nil {
		return err
	}

	parsedUserID, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	user := &User{}
	if err := f.App.DB.Where("telegram_id = ?", args.FromUserID).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user.TelegramID = args.FromUserID
		} else {
			return err
		}
	}
	user.IsAdmin = true
	if err := f.App.DB.Save(user).Error; err != nil {
		return err
	}

	username := "<unknown>"
	if user.Username != "" {
		username = user.Username
	}

	msg := tgbotapi.NewMessage(args.ChatID, fmt.Sprintf("Adding user with id <b>%d</b> (%v) as admin", parsedUserID, username))
	msg.ParseMode = "HTML"
	_, err = f.App.Bot.Send(msg)

	return err
}

func (f *AdminCommand) removeAdminUser(ctx context.Context, args *CommandArguments) error {
	admins := []User{}
	if err := f.App.DB.Where("is_admin = ?", true).Find(&admins).Error; err != nil {
		return err
	}
	suggestions := map[string]string{}
	for _, admin := range admins {
		suggestions[fmt.Sprintf("%v", admin.TelegramID)] = fmt.Sprintf("%d %v", admin.TelegramID, admin.Username)
	}
	userIdStr, err := f.App.AskService.AskForArgument(args.ChatID, "Select admin to remove:\n", suggestions)
	if err != nil {
		return err
	}
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	user := &User{}
	if err := f.App.DB.Where("telegram_id = ?", userId).First(user).Error; err != nil {
		return err
	}
	err = f.App.AskService.Confirm(args.ChatID, fmt.Sprintf("Are you sure you want to remove admin <b>%d</b> (%v)?", userId, user.Username))
	if err != nil {
		return err
	}
	user.IsAdmin = false
	if err := f.App.DB.Save(user).Error; err != nil {
		return err
	}

	return nil
}
