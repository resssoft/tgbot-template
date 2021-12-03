package announcementManager

import "github.com/resssoft/tgbot-template/internal/models"

var commands = []models.TelegramCommand{
	{
		Event: models.AnManagerMenuAdd,
		Name:  "Создать анонс",
		Code:  "",
	},
}
