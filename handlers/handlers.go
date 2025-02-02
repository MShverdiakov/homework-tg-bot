package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"dashka-homework-bot/logger"
	"dashka-homework-bot/storage/mongo"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot             *tgbotapi.BotAPI
	db              *mongo.HomeworkDatabase
	mediaGroups     map[string]string
	mediaGroupsLock sync.Mutex
	lastCleanup     time.Time
}

func NewHandler(bot *tgbotapi.BotAPI, db *mongo.HomeworkDatabase) *Handler {
	return &Handler{
		bot:         bot,
		db:          db,
		mediaGroups: make(map[string]string),
		lastCleanup: time.Now(),
	}
}

func getNextDayName() string {
	tomorrow := time.Now().Add(24 * time.Hour)
	return tomorrow.Weekday().String()
}

func (h *Handler) HandleCommand(message *tgbotapi.Message) {
	// Ensure user is initialized before processing any command
	userID := fmt.Sprintf("%d", message.From.ID)
	username := message.From.UserName

	ctx := context.Background()
	if err := h.ensureUserInitialized(ctx, userID, username); err != nil {
		logger.Error("Error initializing user %s: %v", userID, err)
		h.sendMessage(message.Chat.ID, "Ошибка инициализации, попробуйте позже")
		return
	}

	switch message.Command() {
	case "start":
		nextDay := getNextDayName()
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("Добро пожаловать в Бота для домашних заданий!\n\n"+
				"Чтобы отправить домашку на завтра (%s):\n"+
				"Отправьте снимки с названием предмета в подписи\n"+
				"Пример: 'Математика'\n\n"+
				"2. Родители могут добавлять студентов с помощью команды `/addstudent @username`.\n"+
				"3. Родители могут проверять статус домашнего задания с помощью команды `/checkhw`.\n\n"+
				"Используйте /help, чтобы увидеть все доступные команды.", nextDay))
		h.bot.Send(msg)
	case "help":
		helpText := "📚 *Помощь по Боту для домашних заданий*\n\n" +
			"Вот доступные команды:\n\n" +
			"*/start* - Запустить бота и увидеть инструкции.\n" +
			"*/help* - Показать это сообщение с помощью.\n" +
			"*/addstudent @username* - Добавить студента в ваши контакты (для родителей).\n" +
			"*/checkhw* - Проверить статус домашнего задания ваших студентов (для родителей).\n" +
			"*/schedule* - Посмотреть расписание на завтра.\n\n" +
			"Чтобы отправить домашку:\n" +
			"1. Сделайте фото(снимки) вашего домашнего задания.\n" +
			"2. Добавьте подпись с названием предмета (например, 'Математика').\n" +
			"3. Отправьте фото(снимки) боту.\n\n" +
			"Пример: Отправьте фото с подписью 'Математика', чтобы отправить домашку по математике."
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown" // Использовать форматирование Markdown
		h.bot.Send(msg)
	case "schedule":
		// Get schedule for specific user
		ctx := context.Background()
		nextDay := getNextDayName()
		schedule, err := h.db.GetScheduleForDay(ctx, userID, nextDay)
		if err != nil {
			logger.Error("Error getting schedule for user %s: %v", userID, err)
			h.sendMessage(message.Chat.ID, "Ошибка получения расписания. Попробуйте позже")
			return
		}

		scheduleText := fmt.Sprintf("Завтрашнее (%s) расписание:\n", nextDay)
		for i, subject := range schedule.Subjects {
			scheduleText += fmt.Sprintf("%d. %s\n", i+1, subject.SubjectName)
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, scheduleText)
		h.bot.Send(msg)
	case "addstudent":
		h.handleAddStudent(message)
	case "checkhw":
		h.handleCheckHomework(message)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда. Используйте /help, чтобы увидеть доступные команды.")
		h.bot.Send(msg)
	}
}

func (h *Handler) HandleMessage(message *tgbotapi.Message) {
	h.cleanupMediaGroups()

	// Ensure user is initialized
	userID := fmt.Sprintf("%d", message.From.ID)
	username := message.From.UserName

	ctx := context.Background()
	if err := h.ensureUserInitialized(ctx, userID, username); err != nil {
		logger.Error("Error initializing user %s: %v", userID, err)
		h.sendMessage(message.Chat.ID, "Извините, произошла ошибка при инициализации вашего аккаунта. Пожалуйста, попробуйте позже.")
		return
	}

	if message.Photo == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"Пожалуйста, отправьте снимки вашего домашнего задания и подпишите названием предмета.")
		h.bot.Send(msg)
		return
	}

	caption := message.Caption
	if caption == "" && message.MediaGroupID != "" {
		h.mediaGroupsLock.Lock()
		caption = h.mediaGroups[message.MediaGroupID]
		h.mediaGroupsLock.Unlock()
	}

	if message.MediaGroupID != "" && message.Caption != "" {
		h.mediaGroupsLock.Lock()
		h.mediaGroups[message.MediaGroupID] = message.Caption
		h.mediaGroupsLock.Unlock()
	}

	if caption == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"Пожалуйста, добавьте подпись с названием предмета (например, 'Математика')")
		h.bot.Send(msg)
		return
	}

	nextDay := getNextDayName()
	subject := strings.Title(caption)

	if message.MediaGroupID == "" || message.Caption != "" {
		processingMsg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("Обрабатываю фотографии для %s %s...", nextDay, subject))
		h.bot.Send(processingMsg)
	}

	photoSize := message.Photo[len(message.Photo)-1]

	file, err := h.bot.GetFile(tgbotapi.FileConfig{FileID: photoSize.FileID})
	if err != nil {
		logger.Error("Error getting file: %v", err)
		if message.MediaGroupID == "" {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка обработки фотографий, попробуйте позже")
			h.bot.Send(msg)
		}
		return
	}

	photoBytes, err := downloadFile(file.Link(h.bot.Token))
	if err != nil {
		logger.Error("Error downloading file: %v", err)
		if message.MediaGroupID == "" {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка загрузки фото, попробуйте позже")
			h.bot.Send(msg)
		}
		return
	}

	// Updated to include userID in SaveHomework call
	homeworkID, err := h.db.SaveHomework(
		context.Background(),
		userID,
		nextDay,
		subject,
		photoBytes,
	)
	if err != nil {
		logger.Error("Error saving homework: %v", err)
		if message.MediaGroupID == "" {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка сохранения фото, попробуйте позже")
			h.bot.Send(msg)
		}
		return
	}

	logger.Info("Saved homework with ID: %s for user: %s", homeworkID, userID)

	if message.MediaGroupID == "" || message.Caption != "" {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("Успешно сохранил домашку для %s %s!", nextDay, subject))
		h.bot.Send(msg)
	}
}

func (h *Handler) SetBotCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Запустить бота и увидеть инструкции"},
		{Command: "help", Description: "Показать сообщение с помощью"},
		{Command: "addstudent", Description: "Добавить студента в ваши контакты (для родителей)"},
		{Command: "checkhw", Description: "Проверить статус домашнего задания ваших студентов (для родителей)"},
		{Command: "schedule", Description: "Посмотреть расписание на завтра"},
	}

	config := tgbotapi.NewSetMyCommands(commands...)
	_, err := h.bot.Request(config)
	return err
}

// New helper function to ensure user is initialized
func (h *Handler) ensureUserInitialized(ctx context.Context, userID, username string) error {
	// Create user (this should be idempotent)
	err := h.db.CreateUser(ctx, userID, username)
	if err != nil {
		logger.Error("Error creating user %s: %v", userID, err)
		// Continue anyway as the user might already exist
	}

	// Initialize schedule (this should be idempotent)
	err = h.db.InitializeSchedule(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to initialize schedule: %w", err)
	}

	return nil
}

// Existing helper functions remain the same
func (h *Handler) cleanupMediaGroups() {
	if time.Since(h.lastCleanup) < time.Hour {
		return
	}

	h.mediaGroupsLock.Lock()
	defer h.mediaGroupsLock.Unlock()

	h.mediaGroups = make(map[string]string)
	h.lastCleanup = time.Now()
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (h *Handler) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := h.bot.Send(msg); err != nil {
		logger.Error("Error sending message: %v", err)
	}
}
