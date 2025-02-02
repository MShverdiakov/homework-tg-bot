package handlers

import (
	"context"
	"fmt"
	"strings"

	//"time"
	//"dashka-homework-bot/storage/mongo"
	"dashka-homework-bot/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Add new handler methods
func (h *Handler) handleAddStudent(message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())
	if len(args) < 1 {
		h.sendMessage(message.Chat.ID, "Пожалуйста, укажите Telegram-имя пользователя студента.\nИспользование: /addstudent @username")
		return
	}

	parentUserID := fmt.Sprintf("%d", message.From.ID)
	studentUsername := args[0]

	ctx := context.Background()
	err := h.db.AddStudentContact(ctx, parentUserID, studentUsername)
	if err != nil {
		logger.Error("Error adding student contact for parent %s: %v", parentUserID, err)
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Не удалось добавить студента: %v", err))
		return
	}

	h.sendMessage(message.Chat.ID, fmt.Sprintf("Успешно добавлен студент %s в ваши контакты. Теперь вы можете проверять его домашку с помощью /checkhw", studentUsername))
}

// TODO: can make /homework_status for adult to check on student
func (h *Handler) handleCheckHomework(message *tgbotapi.Message) {
	parentUserID := fmt.Sprintf("%d", message.From.ID)
	ctx := context.Background()

	// Get parent's user document to access their student contacts
	parent, err := h.db.GetParent(ctx, parentUserID)
	if err != nil {
		h.sendMessage(message.Chat.ID, "Не удалось получить вашу информацию. Пожалуйста, попробуйте снова.")
		return
	}

	if len(parent.UserContacts) == 0 {
		h.sendMessage(message.Chat.ID, "Вы еще не добавили ни одного студента. Используйте команду /addstudent @username, чтобы добавить студента.")
		return
	}

	// For each student, check and send homework status
	for _, studentUsername := range parent.UserContacts {
		dayName := getNextDayName() // Using your existing function
		completed, incomplete, homeworks, err := h.db.GetHomeworkStatus(ctx, studentUsername, dayName)
		if err != nil {
			logger.Error("Error checking homework for student %s: %v", studentUsername, err)
			h.sendMessage(message.Chat.ID, fmt.Sprintf("Не удалось проверить домашку для студента %s", studentUsername))
			continue
		}

		// Send status message
		statusMsg := fmt.Sprintf("Статус домашнего задания для %s:\n\n", studentUsername)
		if len(completed) > 0 {
			statusMsg += "✅ Начата домашка:\n"
			for _, subject := range completed {
				statusMsg += fmt.Sprintf("- %s\n", subject)
			}
		}
		if len(incomplete) > 0 {
			statusMsg += "\n❌ Не начата домашка:\n"
			for _, subject := range incomplete {
				statusMsg += fmt.Sprintf("- %s\n", subject)
			}
		}
		h.sendMessage(message.Chat.ID, statusMsg)

		// Send homework photos for completed subjects
		for subject, homeworkList := range homeworks {
			h.sendMessage(message.Chat.ID, fmt.Sprintf("\n📚 Фото домашки для %s:", subject))
			for _, homework := range homeworkList {
				// Create photo message
				photoMsg := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileBytes{
					Name:  "homework.jpg",
					Bytes: homework.Photo,
				})
				photoMsg.Caption = fmt.Sprintf("Предмет: %s\nЗагружено в: %s",
					subject,
					homework.UploadedAt.Format("15:04 02.01.2006"))

				_, err := h.bot.Send(photoMsg)
				if err != nil {
					logger.Error("Error sending homework photo: %v", err)
					h.sendMessage(message.Chat.ID, "Не удалось отправить некоторые фото домашнего задания")
				}
			}
		}
	}
}

// func (h *Handler) StartDailySummaryTask() {
//     go func() {
//         for {
//             now := time.Now()

//             // Skip weekends
//             if strings.Contains(mongo.weekends, now.Weekday().String()) {
//                 logger.Printf("Today is %s (weekend), skipping summary...", now.Weekday().String())
//                 nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 22, 0, 0, 0, now.Location())
//                 time.Sleep(time.Until(nextRun))
//                 continue
//             }

//             // Calculate the next run time (22:00 today or tomorrow)
//             nextRun := time.Date(now.Year(), now.Month(), now.Day(), 22, 0, 0, 0, now.Location())
//             if now.After(nextRun) {
//                 nextRun = nextRun.Add(24 * time.Hour)
//             }

//             // Sleep until the next run time
//             time.Sleep(time.Until(nextRun))

//             // Send daily summaries
//             h.handleCheckHomework()
//         }
//     }()
// }

// func (h *Handler) sendDailySummaries() {
//     ctx := context.Background()

//     // Get all users from the database
//     users, err := h.db.GetAllUsers(ctx)
//     if err != nil {
//         logger.Printf("Error getting users: %v", err)
//         return
//     }

//     // Iterate through users and send summaries to parents
//     for _, user := range users {
//         if user.IsParent {
//             h.sendParentDailySummary(ctx, user)
//         }
//     }
// }
