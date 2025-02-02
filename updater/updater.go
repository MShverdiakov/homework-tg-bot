package updater

import (
	"dashka-homework-bot/logger"

	"dashka-homework-bot/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Updater struct {
	bot      *tgbotapi.BotAPI
	handlers *handlers.Handler
}

func NewUpdater(bot *tgbotapi.BotAPI, handlers *handlers.Handler) *Updater {
	return &Updater{
		bot:      bot,
		handlers: handlers,
	}
}

func (u *Updater) PollUpdates() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := u.bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		// Ignore any non-Message updates
		if update.Message == nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Пожалуйста, отправьте фото(снимки) вашего домашнего задания с подписью, содержащей название предмета.")
			u.bot.Send(msg)
			continue
		}

		logger.Info("[Message from %s] %s", update.Message.From.UserName, update.Message.Text)

		// Check if it's a command
		if update.Message.IsCommand() {
			u.handlers.HandleCommand(update.Message)
			continue
		}

		// Handle media messages
		if update.Message.Photo != nil {
			// Media messages will be handled by HandleMessage
			// It will take care of both single photos and photo albums
			u.handlers.HandleMessage(update.Message)
			continue
		}

		// Handle other text messages
		if update.Message.Text != "" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Пожалуйста, отправьте фото(снимки) вашего домашнего задания с подписью, содержащей название предмета.")
			u.bot.Send(msg)
			continue
		}
	}
}
