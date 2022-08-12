package nighthackbot

import "github.com/alufers/nighthack-bot/dbutil"

type ConfigEntry struct {
	dbutil.Model
	Key   string `gorm:"uniqueindex" json:"key"`
	Value string `json:"value"`
}
