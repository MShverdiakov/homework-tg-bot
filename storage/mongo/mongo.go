package mongo

import (
	"context"
	"dashka-homework-bot/logger"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	defaultTimeout = 10 * time.Second
	summaryTime    = 21
	weekend1       = "Sunday"
	weekend2       = "Tuesday"
)

type User struct {
	UserID       string        `bson:"user_id"`
	Username     string        `bson:"username"`
	Schedule     []DaySchedule `bson:"schedule"`
	CreatedAt    time.Time     `bson:"created_at"`
	UserContacts []string      `bson:"user_contacts"`
	IsParent     bool          `bson:"is_parent"`
}

type HomeworkDatabase struct {
	client   *mongo.Client
	database *mongo.Database
}

type DaySchedule struct {
	DayName  string    `bson:"day_name"`
	Subjects []Subject `bson:"subjects"`
}

type Subject struct {
	SubjectName string     `bson:"subject_name"`
	Homeworks   []Homework `bson:"homeworks"`
}

type Homework struct {
	Photo      []byte    `bson:"photo"`
	UploadedAt time.Time `bson:"uploaded_at"`
	ID         string    `bson:"id"`
	UploadedBy string    `bson:"uploaded_by"`
}

func NewHomeworkDatabase(ctx context.Context, connectionString string) (*HomeworkDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database("homework_tracker")
	logger.Info("Connected to MongoDB successfully")

	return &HomeworkDatabase{
		client:   client,
		database: database,
	}, nil
}

func (m *HomeworkDatabase) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect MongoDB: %w", err)
	}

	logger.Info("Disconnected from MongoDB")
	return nil
}

func (m *HomeworkDatabase) InitializeSchedule(ctx context.Context, userID string) error {
	collection := m.database.Collection("users")

	// First check if user already has a schedule
	var existingUser User
	filter := bson.M{"user_id": userID}
	err := collection.FindOne(ctx, filter).Decode(&existingUser)
	if err == nil && len(existingUser.Schedule) > 0 {
		// User already has a schedule, no need to initialize
		return nil
	}

	schedules := []DaySchedule{
		{
			DayName: "Monday",
			Subjects: []Subject{
				{SubjectName: "Русский", Homeworks: []Homework{}},
				{SubjectName: "История", Homeworks: []Homework{}},
				{SubjectName: "Геометрия", Homeworks: []Homework{}},
				{SubjectName: "Английский", Homeworks: []Homework{}},
				{SubjectName: "ИЗО", Homeworks: []Homework{}},
				{SubjectName: "Литература", Homeworks: []Homework{}},
			},
		},
		// {
		// 	DayName: "Tuesday",
		// 	Subjects: []Subject{
		// 		{SubjectName: "Chemistry", Homeworks: []Homework{}},
		// 		{SubjectName: "Biology", Homeworks: []Homework{}},
		// 		{SubjectName: "English", Homeworks: []Homework{}},
		// 		{SubjectName: "Geography", Homeworks: []Homework{}},
		// 	},
		// },
		{
			DayName: "Wednesday",
			Subjects: []Subject{
				{SubjectName: "Физика", Homeworks: []Homework{}},
				{SubjectName: "Информатика", Homeworks: []Homework{}},
				{SubjectName: "Физкультура", Homeworks: []Homework{}},
				{SubjectName: "Алгебра", Homeworks: []Homework{}},
				{SubjectName: "Английский", Homeworks: []Homework{}},
				{SubjectName: "Общество", Homeworks: []Homework{}},
			},
		},
		{
			DayName: "Thursday",
			Subjects: []Subject{
				{SubjectName: "География", Homeworks: []Homework{}},
				{SubjectName: "Алгебра", Homeworks: []Homework{}},
				{SubjectName: "Биология", Homeworks: []Homework{}},
				{SubjectName: "Вероятность и статистика", Homeworks: []Homework{}},
				{SubjectName: "История", Homeworks: []Homework{}},
				{SubjectName: "Русский", Homeworks: []Homework{}},
				{SubjectName: "Литература", Homeworks: []Homework{}},
				{SubjectName: "Россия мои горизонты", Homeworks: []Homework{}},
			},
		},
		{
			DayName: "Friday",
			Subjects: []Subject{
				{SubjectName: "Труд", Homeworks: []Homework{}},
				{SubjectName: "Физкультура", Homeworks: []Homework{}},
				{SubjectName: "Алгебра", Homeworks: []Homework{}},
				{SubjectName: "Геометрия", Homeworks: []Homework{}},
				{SubjectName: "Английский", Homeworks: []Homework{}},
			},
		},
		{
			DayName: "Saturday",
			Subjects: []Subject{
				{SubjectName: "Физика", Homeworks: []Homework{}},
				{SubjectName: "Алгебра", Homeworks: []Homework{}},
				{SubjectName: "Русский", Homeworks: []Homework{}},
				{SubjectName: "Английский", Homeworks: []Homework{}},
				{SubjectName: "Русский", Homeworks: []Homework{}},
				{SubjectName: "География", Homeworks: []Homework{}},
				{SubjectName: "Музыка", Homeworks: []Homework{}},
			},
		},
	}

	update := bson.M{
		"$set": bson.M{
			"schedule": schedules,
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to initialize schedule for user %s: %w", userID, err)
	}

	logger.Info("Schedule initialization result for user %s: matched=%d, modified=%d, upserted=%v",
		userID, result.MatchedCount, result.ModifiedCount, result.UpsertedID != nil)

	return nil
}

func (m *HomeworkDatabase) CreateUser(ctx context.Context, userID, username string) error {
	collection := m.database.Collection("users")

	// Check if user already exists
	var existingUser User
	filter := bson.M{"user_id": userID}
	err := collection.FindOne(ctx, filter).Decode(&existingUser)
	if err == nil {
		// User already exists, just update the username if it changed
		if existingUser.Username != username {
			update := bson.M{
				"$set": bson.M{"username": username},
			}
			_, err = collection.UpdateOne(ctx, filter, update)
			if err != nil {
				return fmt.Errorf("failed to update username: %w", err)
			}
		}
		return nil
	}

	// Only create new user if they don't exist
	if err == mongo.ErrNoDocuments {
		user := User{
			UserID:    userID,
			Username:  username,
			CreatedAt: time.Now(),
			Schedule:  []DaySchedule{}, // Empty schedule, will be initialized separately
		}

		_, err = collection.InsertOne(ctx, user)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		return nil
	}

	return fmt.Errorf("error checking for existing user: %w", err)
}

func (m *HomeworkDatabase) GetUser(ctx context.Context, userID string) (*User, error) {
	collection := m.database.Collection("users")

	var user User
	filter := bson.M{"user_id": userID}
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to find user %s: %w", userID, err)
	}

	return &user, nil
}

func (m *HomeworkDatabase) GetScheduleForDay(ctx context.Context, userID, day string) (*DaySchedule, error) {
	collection := m.database.Collection("users")

	var user User
	filter := bson.M{"user_id": userID}
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to find user %s: %w", userID, err)
	}

	for _, schedule := range user.Schedule {
		if schedule.DayName == day {
			return &schedule, nil
		}
	}

	return nil, fmt.Errorf("no schedule found for day %s", day)
}

func (m *HomeworkDatabase) SaveHomework(ctx context.Context, userID, dayName, subjectName string, photoData []byte) (string, error) {
	collection := m.database.Collection("users")

	homeworkID := fmt.Sprintf("%s-%s-%s-%d", userID, dayName, subjectName, time.Now().Unix())

	homework := Homework{
		Photo:      photoData,
		UploadedAt: time.Now(),
		ID:         homeworkID,
		UploadedBy: userID,
	}

	filter := bson.M{
		"user_id": userID,
		"schedule": bson.M{
			"$elemMatch": bson.M{
				"day_name": dayName,
				"subjects.subject_name": bson.M{
					"$regex":   subjectName,
					"$options": "i",
				},
			},
		},
	}

	update := bson.M{
		"$push": bson.M{
			"schedule.$[day].subjects.$[subject].homeworks": homework,
		},
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"day.day_name": dayName},
			bson.M{"subject.subject_name": bson.M{"$regex": subjectName, "$options": "i"}},
		},
	})

	result, err := collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return "", fmt.Errorf("failed to save homework: %w", err)
	}

	if result.MatchedCount == 0 {
		return "", fmt.Errorf("no matching user/day/subject found for %s/%s/%s", userID, dayName, subjectName)
	}

	return homeworkID, nil
}

func (m *HomeworkDatabase) GetHomework(ctx context.Context, userID, dayName, subjectName string, homeworkID string) (*Homework, error) {
	collection := m.database.Collection("users")

	filter := bson.M{
		"user_id": userID,
		"schedule": bson.M{
			"$elemMatch": bson.M{
				"day_name": dayName,
			},
		},
	}

	var user User
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no user found with ID %s", userID)
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	for _, schedule := range user.Schedule {
		if schedule.DayName == dayName {
			for _, subject := range schedule.Subjects {
				if subject.SubjectName == subjectName {
					for _, homework := range subject.Homeworks {
						if homework.ID == homeworkID {
							return &homework, nil
						}
					}
					return nil, fmt.Errorf("homework with ID %s not found", homeworkID)
				}
			}
			return nil, fmt.Errorf("subject %s not found for day %s", subjectName, dayName)
		}
	}

	return nil, fmt.Errorf("day %s not found in schedule", dayName)
}

func (m *HomeworkDatabase) GetAllHomework(ctx context.Context, userID, dayName, subjectName string) ([]Homework, error) {
	collection := m.database.Collection("users")

	filter := bson.M{
		"user_id": userID,
		"schedule": bson.M{
			"$elemMatch": bson.M{
				"day_name": dayName,
			},
		},
	}

	var user User
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no user found with ID %s", userID)
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	for _, schedule := range user.Schedule {
		if schedule.DayName == dayName {
			for _, subject := range schedule.Subjects {
				if subject.SubjectName == subjectName {
					return subject.Homeworks, nil
				}
			}
			return nil, fmt.Errorf("subject %s not found for day %s", subjectName, dayName)
		}
	}

	return nil, fmt.Errorf("day %s not found in schedule", dayName)
}

func (m *HomeworkDatabase) GetParent(ctx context.Context, parentUserID string) (*User, error) {
	collection := m.database.Collection("users")

	var parent User
	err := collection.FindOne(ctx, bson.M{"user_id": parentUserID}).Decode(&parent)
	if err != nil {
		logger.Error("Error getting parent user %s: %v", parentUserID, err)
		return nil, err
	}

	return &parent, nil
}

func (m *HomeworkDatabase) AddStudentContact(ctx context.Context, parentUserID, studentUsername string) error {
	collection := m.database.Collection("users")

	// Verify and format student username
	if !strings.HasPrefix(studentUsername, "@") {
		studentUsername = "@" + studentUsername
	}

	// First, verify that the student exists in our database
	var studentUser User
	err := collection.FindOne(ctx, bson.M{"username": strings.TrimPrefix(studentUsername, "@")}).Decode(&studentUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("student with username %s not found", studentUsername)
		}
		return fmt.Errorf("failed to verify student: %v", err)
	}

	// Add student to parent's contacts
	update := bson.M{
		"$addToSet": bson.M{
			"user_contacts": studentUsername,
		},
		"$set": bson.M{
			"is_parent": true,
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"user_id": parentUserID}, update)
	if err != nil {
		return fmt.Errorf("failed to add student contact: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no user found with ID %s", parentUserID)
	}

	return nil
}

func (m *HomeworkDatabase) GetHomeworkStatus(ctx context.Context, studentUsername, dayName string) ([]string, []string, map[string][]Homework, error) {
	collection := m.database.Collection("users")

	// Find the student by username
	var student User
	err := collection.FindOne(ctx, bson.M{"username": strings.TrimPrefix(studentUsername, "@")}).Decode(&student)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get student: %v", err)
	}

	var completedSubjects []string
	var incompleteSubjects []string
	homeworkMap := make(map[string][]Homework)

	// Find schedule for the specified day
	for _, schedule := range student.Schedule {
		if schedule.DayName == dayName {
			for _, subject := range schedule.Subjects {
				if len(subject.Homeworks) > 0 {
					completedSubjects = append(completedSubjects, subject.SubjectName)
					homeworkMap[subject.SubjectName] = subject.Homeworks
				} else {
					incompleteSubjects = append(incompleteSubjects, subject.SubjectName)
				}
			}
			break
		}
	}

	return completedSubjects, incompleteSubjects, homeworkMap, nil
}

func (m *HomeworkDatabase) EraseHomeworkForOldDay(ctx context.Context, userID, dayName string) error {
	collection := m.database.Collection("users")

	filter := bson.M{
		"user_id":           userID,
		"schedule.day_name": dayName,
	}

	update := bson.M{
		"$set": bson.M{
			"schedule.$[day].subjects.$[].homeworks": []interface{}{},
		},
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"day.day_name": dayName},
		},
	})

	result, err := collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("failed to erase homework for user %s, day %s: %w", userID, dayName, err)
	}

	if result.MatchedCount == 0 {
		logger.Error("No schedule found for user: %s, day: %s", userID, dayName)
	}

	return nil
}

func (m *HomeworkDatabase) StartEraseAtMidnight() {
	// This method would need to be modified to handle all users
	for {
		now := time.Now()
		oldDay := now.AddDate(0, 0, -5).Weekday().String()

		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

		// Get all users and erase for each
		collection := m.database.Collection("users")
		cursor, err := collection.Find(ctx, bson.M{})
		if err != nil {
			logger.Error("Error finding users: %v", err)
			cancel()
			continue
		}

		var users []User
		if err := cursor.All(ctx, &users); err != nil {
			logger.Error("Error decoding users: %v", err)
			cancel()
			continue
		}

		for _, user := range users {
			if err := m.EraseHomeworkForOldDay(ctx, user.UserID, oldDay); err != nil {
				logger.Error("Error erasing homework for user %s, day %s: %v", user.UserID, oldDay, err)
			}
		}

		cancel()

		// Calculate time until midnight
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(time.Until(nextMidnight))
	}
}

func (m *HomeworkDatabase) SendDailySummaries(bot *tgbotapi.BotAPI) error {
	ctx := context.Background()
	collection := m.database.Collection("users")

	// Find all parent users
	filter := bson.M{"is_parent": true}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find parents: %w", err)
	}
	defer cursor.Close(ctx)

	nextDay := time.Now().Add(24 * time.Hour).Weekday().String()
	// Skip weekends
	if nextDay == weekend1 || nextDay == weekend2 {
		logger.Info("Skipping summary notifications for %s", nextDay)
		return nil
	}

	var parents []User
	if err := cursor.All(ctx, &parents); err != nil {
		return fmt.Errorf("failed to decode parents: %w", err)
	}

	for _, parent := range parents {
		// For each parent's student contacts
		for _, studentUsername := range parent.UserContacts {
			completed, incomplete, homeworks, err := m.GetHomeworkStatus(ctx, studentUsername, nextDay)
			if err != nil {
				logger.Error("Error getting homework status for student %s: %v", studentUsername, err)
				continue
			}

			// Create summary message
			summaryMsg := fmt.Sprintf("Статус домашнего задания для %s:\n\n", studentUsername)

			if len(completed) > 0 {
				summaryMsg += "✅ Начата домашка:\n"
				for _, subject := range completed {
					summaryMsg += fmt.Sprintf("- %s\n", subject)
				}
			}

			if len(incomplete) > 0 {
				summaryMsg += "\n❌ Не начата домашка:\n"
				for _, subject := range incomplete {
					summaryMsg += fmt.Sprintf("- %s\n", subject)
				}
			}

			// Convert parent.UserID to int64 for telegram API
			parentID, err := strconv.ParseInt(parent.UserID, 10, 64)
			if err != nil {
				logger.Error("Error converting parent ID: %v", err)
				continue
			}

			// Send text summary
			msg := tgbotapi.NewMessage(parentID, summaryMsg)
			if _, err := bot.Send(msg); err != nil {
				logger.Error("Error sending summary to parent %s: %v", parent.UserID, err)
				continue
			}

			// Send homework photos
			for subject, homeworkList := range homeworks {
				photoMsg := fmt.Sprintf("\n📚 Сегодняшняя домашка по %s:", subject)
				msg := tgbotapi.NewMessage(parentID, photoMsg)
				bot.Send(msg)

				for _, homework := range homeworkList {
					photo := tgbotapi.NewPhoto(parentID, tgbotapi.FileBytes{
						Name:  "homework.jpg",
						Bytes: homework.Photo,
					})
					photo.Caption = fmt.Sprintf("Предмет: %s\nЗагружено в: %s",
						subject,
						homework.UploadedAt.Format("15:04 02.01.2006"))

					if _, err := bot.Send(photo); err != nil {
						logger.Error("Error sending homework photo: %v", err)
					}
				}
			}
		}
	}

	return nil
}

// Add this method to start the daily summary scheduler
func (m *HomeworkDatabase) StartDailySummaries(bot *tgbotapi.BotAPI) {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), summaryTime, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			// Wait until next scheduled time
			time.Sleep(time.Until(next))

			// Send summaries
			if err := m.SendDailySummaries(bot); err != nil {
				logger.Error("Error sending daily summaries: %v", err)
			}
		}
	}()
}
