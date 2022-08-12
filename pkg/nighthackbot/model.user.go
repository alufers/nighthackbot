package nighthackbot

import "github.com/alufers/nighthack-bot/dbutil"

type User struct {
	dbutil.Model
	TelegramID          int64   `gorm:"uniqueindex" json:"telegramID"`
	Username            string  `json:"username"`
	Email               *string `json:"email"`
	IsAdmin             bool    `json:"isAdmin"`
	PingAboutNighthacks bool    `json:"pingAboutNighthacks"`
}
