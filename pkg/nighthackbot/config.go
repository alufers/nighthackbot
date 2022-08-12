package nighthackbot

type Config struct {
	Telegram struct {
		Token string `mapstructure:"token"`
		Debug bool   `mapstructure:"debug"`
	} `mapstructure:"telegram"`
	DB struct {
		Type     string `mapstructure:"type"`
		DSN      string `mapstructure:"dsn"`      // postgres
		Filename string `mapstructure:"filename"` // sqlite
	} `mapstructure:"db"`
}
