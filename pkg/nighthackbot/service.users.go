package nighthackbot

import (
	"errors"

	"gorm.io/gorm"
)

type UsersService struct {
	BotApp *BotApp
}

func NewUsersService(botApp *BotApp) *UsersService {
	return &UsersService{
		BotApp: botApp,
	}
}

func (s *UsersService) AddUserToArgs(args *CommandArguments) error {
	user := &User{}

	// get the user from db or create a new one
	if err := s.BotApp.DB.Where("telegram_id = ?", args.FromUserID).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user.TelegramID = args.FromUserID
			user.Username = args.FromUserName

			if err := s.BotApp.DB.Save(user).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if user.Username != args.FromUserName {
		user.Username = args.FromUserName
		if err := s.BotApp.DB.Save(user).Error; err != nil {
			return err
		}
	}
	args.User = user

	return nil
}
