package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

// Функция для обновления сообщения в JSON
func updateMessageInJSON(filename string, messageID int, newText string, newPhotoIDs []string, newPhotoPaths []string) error {
	messagesData, err := loadMessagesData(filename)
	if err != nil {
		return fmt.Errorf("ошибка загрузки данных сообщений: %v", err)
	}

	found := false
	for i, msg := range messagesData.Messages {
		if msg.MessageID == messageID {
			messagesData.Messages[i].Text = newText
			if len(newPhotoIDs) > 0 {
				messagesData.Messages[i].PhotoIDs = newPhotoIDs
			}
			if len(newPhotoPaths) > 0 {
				messagesData.Messages[i].PhotoPaths = newPhotoPaths
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("сообщение с ID %d не найдено", messageID)
	}

	if err := saveMessagesData(filename, messagesData); err != nil {
		return fmt.Errorf("ошибка сохранения данных: %v", err)
	}

	return nil
}

// Обработчик редактирования сообщений
func handleEditedMessage(bot *tgbotapi.BotAPI, usersDataFile, messagesDataFile string, editedMessage *tgbotapi.Message) {
	newText := editedMessage.Text
	var newPhotoIDs []string
	var newPhotoPaths []string

	if editedMessage.Photo != nil && len(editedMessage.Photo) > 0 {
		photo := editedMessage.Photo[len(editedMessage.Photo)-1]
		newPhotoIDs = append(newPhotoIDs, photo.FileID)

		photoPath, err := saveMessagePhoto(bot.Token, photo.FileID)
		if err != nil {
			log.Printf("Ошибка при сохранении нового фото: %v", err)
		} else {
			newPhotoPaths = append(newPhotoPaths, photoPath)
		}
	}

	err := updateMessageInJSON(messagesDataFile, editedMessage.MessageID, newText, newPhotoIDs, newPhotoPaths)
	if err != nil {
		log.Printf("Ошибка обновления сообщения: %v", err)
		return
	}

	log.Printf("Сообщение с ID %d успешно обновлено", editedMessage.MessageID)
}

// Функция для получения аватара пользователя
func getUserAvatar(botToken string, userID string) (string, error) {
	photosURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUserProfilePhotos?user_id=%s", botToken, userID)
	resp, err := http.Get(photosURL)
	if err != nil {
		return "", fmt.Errorf("ошибка при запросе к API: %v", err)
	}
	defer resp.Body.Close()

	var userProfilePhotos UserProfilePhotos
	err = json.NewDecoder(resp.Body).Decode(&userProfilePhotos)
	if err != nil {
		return "", fmt.Errorf("ошибка при декодировании JSON: %v", err)
	}

	if userProfilePhotos.Result.TotalCount == 0 {
		return "у пользователя нет аватара", nil
	}

	fileID := userProfilePhotos.Result.Photos[0][0].FileID
	fileInfoURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", botToken, fileID)
	resp, err = http.Get(fileInfoURL)
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

	filename := fmt.Sprintf("photos/ava_%s.jpg", userID)
	out, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("ошибка при создании файла: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка при записи файла: %v", err)
	}

	return fmt.Sprintf("аватар успешно скачан как %s", filename), nil
}

// Обработка редактирования сообщений
func handleMessageEdit(bot *tgbotapi.BotAPI, usersDataFile, messagesDataFile string, update tgbotapi.Update) {
	if update.EditedMessage != nil {
		handleEditedMessage(bot, usersDataFile, messagesDataFile, update.EditedMessage)
	}
}

// Обработка пользователя
func handleUser(usersData UsersData, userID int64, userFirstName, userLastName, username string) (UsersData, bool, int) {
	userExists := false
	var existingUserIndex int

	for i, user := range usersData.Users {
		if user.UserID == userID {
			userExists = true
			existingUserIndex = i
			break
		}
	}

	return usersData, userExists, existingUserIndex
}

// Обработка контакта
func handleContact(bot *tgbotapi.BotAPI, usersData UsersData, update tgbotapi.Update,
	userExists bool, existingUserIndex int, userID int64,
	userFirstName, userLastName, username string) UsersData {

	phoneNumber := update.Message.Contact.PhoneNumber

	if userExists {
		usersData.Users[existingUserIndex].PhoneNumber = phoneNumber
	} else {
		newUser := User{
			UserID:           userID,
			UserFirstName:    userFirstName,
			UserLastName:     userLastName,
			Username:         username,
			PhoneNumber:      phoneNumber,
			RegistrationDate: time.Now(),
		}
		usersData.Users = append(usersData.Users, newUser)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Номер телефона сохранен!")
	msg.ReplyToMessageID = update.Message.MessageID
	bot.Send(msg)

	return usersData
}

// Добавление нового пользователя, если его нет
func addNewUserIfNotExists(usersData UsersData, userExists bool, userID int64,
	userFirstName, userLastName, username string) UsersData {

	if !userExists {
		newUser := User{
			UserID:           userID,
			UserFirstName:    userFirstName,
			UserLastName:     userLastName,
			Username:         username,
			RegistrationDate: time.Now(),
		}
		usersData.Users = append(usersData.Users, newUser)
	}

	return usersData
}

// Обработка сообщения
func handleMessage(messagesData MessagesData, userID int64, message *tgbotapi.Message) MessagesData {
	newMessage := Message{
		UserID:      userID,
		MessageID:   message.MessageID,
		Text:        message.Text,
		MessageDate: time.Now(),
	}

	messagesData.Messages = append(messagesData.Messages, newMessage)

	return messagesData
}

// Обработка фото
func handlePhoto(botToken string, messagesData MessagesData, message *tgbotapi.Message) (MessagesData, error) {
	if message.Photo != nil && len(message.Photo) > 0 {
		var photoIDs []string
		var photoPaths []string

		photo := message.Photo[len(message.Photo)-1]
		photoIDs = append(photoIDs, photo.FileID)

		photoPath, err := saveMessagePhoto(botToken, photo.FileID)
		if err != nil {
			return messagesData, fmt.Errorf("ошибка при сохранении фото: %v", err)
		}

		photoPaths = append(photoPaths, photoPath)

		if len(messagesData.Messages) > 0 {
			lastIdx := len(messagesData.Messages) - 1
			messagesData.Messages[lastIdx].PhotoIDs = photoIDs
			messagesData.Messages[lastIdx].PhotoPaths = photoPaths
		}
	}

	return messagesData, nil
}

// Формирование и отправка ответа
func sendReply(bot *tgbotapi.BotAPI, message *tgbotapi.Message, userID int64,
	userFirstName, userLastName, username string, photoAttached bool) {

	avatarInfo, err := getUserAvatar(bot.Token, strconv.FormatInt(userID, 10))

	replyText := "Получены следующие данные:\n\n"
	replyText += "Текст сообщения: " + message.Text + "\n"
	replyText += "ID сообщения: " + strconv.Itoa(message.MessageID) + "\n"
	replyText += "Имя пользователя: " + userFirstName + "\n"
	replyText += "Фамилия пользователя: " + userLastName + "\n"
	replyText += "Никнейм пользователя: " + username + "\n"
	replyText += "ID пользователя: " + strconv.FormatInt(userID, 10) + "\n"

	if photoAttached {
		replyText += "К сообщению прикреплено фото\n"
	}

	if err != nil {
		replyText += "\nИнформация об аватаре: " + err.Error()
	} else {
		replyText += "\nИнформация об аватаре: " + avatarInfo
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}

func main() {
	// Устанавливаем токен бота
	botToken := "7217078454:AAGqrgEr_JuoJnwqwf1xU5P3lO--GnDtCIg"

	// Создаем новый экземпляр бота API
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		// Если произошла ошибка при создании бота, завершаем программу
		log.Panic(err)
	}

	// Отключаем режим отладки для бота
	bot.Debug = false

	// Выводим в лог информацию о том, что бот успешно подключился
	log.Printf("Подключился как %s", bot.Self.UserName)

	// Создаем новый канал обновлений с таймаутом 10 секунд
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10
	updates := bot.GetUpdatesChan(u)

	// Указываем пути к файлам данных
	usersDataFile := "JSON/users_data.json"       // Файл для хранения данных пользователей
	messagesDataFile := "JSON/messages_data.json" // Файл для хранения сообщений

	// Начинаем бесконечный цикл обработки обновлений
	for update := range updates {
		// Обрабатываем редактирование сообщений
		handleMessageEdit(bot, usersDataFile, messagesDataFile, update)

		// Пропускаем обновления без сообщений
		if update.Message == nil {
			continue
		}

		// Загружаем текущие данные пользователей и сообщений
		usersData, _ := loadUsersData(usersDataFile)
		messagesData, _ := loadMessagesData(messagesDataFile)

		// Получаем информацию о пользователе из сообщения
		userID := update.Message.From.ID
		userFirstName := update.Message.From.FirstName
		userLastName := update.Message.From.LastName
		username := update.Message.From.UserName

		// Проверяем, существует ли уже пользователь в базе данных
		usersData, userExists, existingUserIndex := handleUser(usersData, userID, userFirstName, userLastName, username)

		// Если сообщение содержит контактную информацию
		if update.Message.Contact != nil {
			// Обрабатываем контакт и сохраняем данные пользователя
			usersData = handleContact(bot, usersData, update, userExists, existingUserIndex,
				userID, userFirstName, userLastName, username)
			saveUsersData(usersDataFile, usersData)
			continue
		}

		// Добавляем нового пользователя, если его еще нет в базе
		usersData = addNewUserIfNotExists(usersData, userExists, userID, userFirstName, userLastName, username)
		saveUsersData(usersDataFile, usersData)

		// Обрабатываем текстовое сообщение
		messagesData = handleMessage(messagesData, userID, update.Message)

		// Проверяем, есть ли в сообщении фото
		photoAttached := update.Message.Photo != nil && len(update.Message.Photo) > 0
		if photoAttached {
			// Если есть фото, обрабатываем и сохраняем его
			var err error
			messagesData, err = handlePhoto(bot.Token, messagesData, update.Message)
			if err != nil {
				log.Println(err)
			}
		}

		// Сохраняем обновленные данные сообщений
		saveMessagesData(messagesDataFile, messagesData)

		// Формируем и отправляем ответное сообщение пользователю
		sendReply(bot, update.Message, userID, userFirstName, userLastName, username, photoAttached)
	}
}
