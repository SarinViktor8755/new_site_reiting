package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Структуры для парсинга JSON ответов от Telegram API
type UserProfilePhotos struct {
	Result struct {
		TotalCount int `json:"total_count"`
		Photos     [][]struct {
			FileID string `json:"file_id"`
		} `json:"photos"`
	} `json:"result"`
}

type FileResponse struct {
	Result struct {
		FilePath string `json:"file_path"`
	} `json:"result"`
}

// Структура для хранения информации о пользователях
type User struct {
	UserID           int64     `json:"userID"`
	UserFirstName    string    `json:"userFirstName"`
	UserLastName     string    `json:"userLastName,omitempty"`
	Username         string    `json:"username,omitempty"`
	PhoneNumber      string    `json:"phoneNumber,omitempty"`
	RegistrationDate time.Time `json:"registrationDate"`
}

type UsersData struct {
	Users []User `json:"users"`
}

// Структуры для хранения сообщений
type Message struct {
	UserID      int64     `json:"userID"`
	MessageID   int       `json:"messageID"`
	Text        string    `json:"text,omitempty"`
	PhotoIDs    []string  `json:"photoIDs,omitempty"`
	PhotoPaths  []string  `json:"photoPaths,omitempty"`
	Distance    float64   `json:"distance"` // Новое поле для хранения дистанции
	MessageDate time.Time `json:"messageDate"`
}

type MessagesData struct {
	Messages []Message `json:"messages"`
}

// Функции для работы с пользователями
func loadUsersData(filename string) (UsersData, error) {
	var usersData UsersData
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return UsersData{Users: []User{}}, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return UsersData{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&usersData)
	if err != nil {
		return UsersData{}, err
	}
	return usersData, nil
}

func saveUsersData(filename string, usersData UsersData) error {
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(usersData)
	if err != nil {
		return err
	}
	return nil
}

// Функции для работы с сообщениями
func loadMessagesData(filename string) (MessagesData, error) {
	var messagesData MessagesData
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return MessagesData{Messages: []Message{}}, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return MessagesData{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&messagesData)
	if err != nil {
		return MessagesData{}, err
	}
	return messagesData, nil
}

func saveMessagesData(filename string, messagesData MessagesData) error {
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(messagesData)
	if err != nil {
		return err
	}
	return nil
}

// Функция для сохранения фото из сообщения
func saveMessagePhoto(botToken string, fileID string) (string, error) {
	fileInfoURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", botToken, fileID)
	resp, err := http.Get(fileInfoURL)
	if err != nil {
		return "", fmt.Errorf("ошибка при запросе к getFile: %v", err)
	}
	defer resp.Body.Close()

	var fileResponse FileResponse
	err = json.NewDecoder(resp.Body).Decode(&fileResponse)
	if err != nil {
		return "", fmt.Errorf("ошибка при декодировании JSON для getFile: %v", err)
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, fileResponse.Result.FilePath)
	if _, err := os.Stat("photos"); os.IsNotExist(err) {
		os.Mkdir("photos", 0755)
	}

	resp, err = http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("ошибка при скачивании файла: %v", err)
	}
	defer resp.Body.Close()

	filename := fmt.Sprintf("photos/%s_%d.jpg", fileID, time.Now().Unix())
	out, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("ошибка при создании файла: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка при записи файла: %v", err)
	}

	return filename, nil
}

// Извлечение дистанции из сообщения
func extractDistance(message string) float64 {
	cleaned := strings.ReplaceAll(message, " ", "")
	reKm := regexp.MustCompile(`(?i)#км`)
	if !reKm.MatchString(cleaned) {
		return -1
	}
	reNumbers := regexp.MustCompile(`\+(\d+[\.,]?\d*)`)
	matches := reNumbers.FindAllStringSubmatch(cleaned, -1)
	if len(matches) == 0 {
		return -1
	}
	var sum float64
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		numStr := strings.Replace(match[1], ",", ".", -1)
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			continue
		}
		sum += num
	}
	return sum
}

func FloatToString(f float64) string {
	str := strconv.FormatFloat(f, 'f', 10, 64)
	str = strings.TrimRight(str, "0")
	str = strings.TrimRight(str, ".")
	return str
}

func setupRESTAPI(usersDataFile, messagesDataFile string) {
	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		usersData, err := loadUsersData(usersDataFile)
		if err != nil {
			http.Error(w, "Ошибка загрузки данных пользователей", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(usersData)
	})

	http.HandleFunc("/api/messages", func(w http.ResponseWriter, r *http.Request) {
		messagesData, err := loadMessagesData(messagesDataFile)
		if err != nil {
			http.Error(w, "Ошибка загрузки данных сообщений", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesData)
	})

	http.HandleFunc("/api/messages/month", func(w http.ResponseWriter, r *http.Request) {
		yearStr := r.URL.Query().Get("year")
		monthStr := r.URL.Query().Get("month")

		if yearStr == "" || monthStr == "" {
			http.Error(w, "Необходимо указать параметры 'year' и 'month'", http.StatusBadRequest)
			return
		}

		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 1900 || year > 2100 {
			http.Error(w, "Неверный формат года", http.StatusBadRequest)
			return
		}

		month, err := strconv.Atoi(monthStr)
		if err != nil || month < 1 || month > 12 {
			http.Error(w, "Неверный формат месяца", http.StatusBadRequest)
			return
		}

		messagesData, err := loadMessagesData(messagesDataFile)
		if err != nil {
			http.Error(w, "Ошибка загрузки данных сообщений", http.StatusInternalServerError)
			return
		}

		var filteredMessages []Message
		for _, msg := range messagesData.Messages {
			if msg.MessageDate.Year() == year && int(msg.MessageDate.Month()) == month {
				filteredMessages = append(filteredMessages, msg)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MessagesData{Messages: filteredMessages})
	})

	log.Println("REST API запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleEditedMessage(bot *tgbotapi.BotAPI, usersDataFile, messagesDataFile string, editedMessage *tgbotapi.Message) {
	newText := editedMessage.Text
	if newText == "" {
		newText = editedMessage.Caption
	}

	distance := extractDistance(newText)
	if distance <= 0 {
		log.Printf("Отредактированное сообщение #%d не содержит информации о пробежке", editedMessage.MessageID)
		return
	}

	var photoIDs []string
	var photoPaths []string
	if editedMessage.Photo != nil && len(editedMessage.Photo) > 0 {
		photo := editedMessage.Photo[len(editedMessage.Photo)-1]
		photoIDs = append(photoIDs, photo.FileID)
		photoPath, err := saveMessagePhoto(bot.Token, photo.FileID)
		if err != nil {
			log.Printf("Ошибка при сохранении нового фото для сообщения #%d: %v", editedMessage.MessageID, err)
			return
		}
		photoPaths = append(photoPaths, photoPath)
	}

	messagesData, _ := loadMessagesData(messagesDataFile)
	for i, msg := range messagesData.Messages {
		if msg.MessageID == editedMessage.MessageID {
			messagesData.Messages[i].Text = newText
			messagesData.Messages[i].PhotoIDs = photoIDs
			messagesData.Messages[i].PhotoPaths = photoPaths
			messagesData.Messages[i].Distance = distance
			break
		}
	}
	saveMessagesData(messagesDataFile, messagesData)

	replyText := fmt.Sprintf("Ваша пробежка обновлена: %s км", FloatToString(distance))
	msg := tgbotapi.NewMessage(editedMessage.Chat.ID, replyText)
	bot.Send(msg)
}

func main() {
	botToken := "7217078454:AAGqrgEr_JuoJnwqwf1xU5P3lO--GnDtCIg"
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("Подключился как %s", bot.Self.UserName)

	usersDataFile := "JSON/users_data.json"
	messagesDataFile := "JSON/messages_data.json"

	go setupRESTAPI(usersDataFile, messagesDataFile)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil && strings.HasPrefix(update.Message.Text, "/rm") {
			parts := strings.SplitN(update.Message.Text, "_", 2)
			if len(parts) < 2 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат команды. Используйте: /rm_<ID>")
				bot.Send(msg)
				continue
			}

			messageIDStr := parts[1]
			messageID, err := strconv.Atoi(messageIDStr)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный ID сообщения")
				bot.Send(msg)
				continue
			}

			messagesData, _ := loadMessagesData(messagesDataFile)
			var found bool
			for i, msg := range messagesData.Messages {
				if msg.MessageID == messageID && msg.UserID == update.Message.From.ID {
					messagesData.Messages = append(messagesData.Messages[:i], messagesData.Messages[i+1:]...)
					saveMessagesData(messagesDataFile, messagesData)
					replyText := fmt.Sprintf("Сообщение #%d удалено:\n%s", messageID, msg.Text)
					response := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
					bot.Send(response)
					found = true
					break
				}
			}

			if !found {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Сообщение не найдено")
				bot.Send(msg)
			}
			continue
		}

		if update.EditedMessage != nil {
			handleEditedMessage(bot, usersDataFile, messagesDataFile, update.EditedMessage)
			continue
		}

		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		userFirstName := update.Message.From.FirstName
		userLastName := update.Message.From.LastName
		username := update.Message.From.UserName

		usersData, _ := loadUsersData(usersDataFile)
		userExists := false
		for _, user := range usersData.Users {
			if user.UserID == userID {
				userExists = true
				break
			}
		}

		if !userExists {
			newUser := User{
				UserID:           userID,
				UserFirstName:    userFirstName,
				UserLastName:     userLastName,
				Username:         username,
				RegistrationDate: time.Now(),
			}
			usersData.Users = append(usersData.Users, newUser)
			saveUsersData(usersDataFile, usersData)
		}

		text := update.Message.Text
		if text == "" {
			text = update.Message.Caption
		}

		distance := extractDistance(text)
		if distance <= 0 {
			continue
		}

		if update.Message.Photo == nil || len(update.Message.Photo) == 0 {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Прикрепите скрин трека пробежки")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
			continue
		}

		photo := update.Message.Photo[len(update.Message.Photo)-1]
		photoPath, err := saveMessagePhoto(bot.Token, photo.FileID)
		if err != nil {
			log.Printf("Ошибка при сохранении фото: %v", err)
			continue
		}

		newMessage := Message{
			UserID:      userID,
			MessageID:   update.Message.MessageID,
			Text:        text,
			PhotoIDs:    []string{photo.FileID},
			PhotoPaths:  []string{photoPath},
			Distance:    distance,
			MessageDate: time.Now(),
		}

		messagesData, _ := loadMessagesData(messagesDataFile)
		messagesData.Messages = append(messagesData.Messages, newMessage)
		saveMessagesData(messagesDataFile, messagesData)

		replyText := fmt.Sprintf("Засчитано: %s км\nСтатистика на сайте: asdasd.com\nЕсли вы ошиблись, удалите это сообщение командой:\n/rm_%d",
			FloatToString(distance), update.Message.MessageID)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
		bot.Send(msg)
	}
}
