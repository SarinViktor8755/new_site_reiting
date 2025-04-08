package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
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
    UserID          int64     `json:"userID"`
    UserFirstName   string    `json:"userFirstName"`
    UserLastName    string    `json:"userLastName,omitempty"`
    Username        string    `json:"username,omitempty"`
    PhoneNumber     string    `json:"phoneNumber,omitempty"`
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
    resp, err = http.Get(fileURL)
    if err != nil {
        return "", fmt.Errorf("ошибка при скачивании файла: %v", err)
    }
    defer resp.Body.Close()

    filename := fmt.Sprintf("avatars/%s.jpg", userID)
    if _, err := os.Stat("avatars"); os.IsNotExist(err) {
        os.Mkdir("avatars", 0755)
    }

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

// Обработка пользователя
func handleUser(update tgbotapi.Update, usersData *UsersData) bool {
    userID := update.Message.From.ID
    for _, user := range usersData.Users {
        if user.UserID == userID {
            return true // Пользователь уже существует
        }
    }
    return false // Пользователь новый
}

// Обработка контакта
func handleContact(update tgbotapi.Update, usersData *UsersData, bot *tgbotapi.BotAPI) {
    userID := update.Message.From.ID
    phoneNumber := update.Message.Contact.PhoneNumber
    userExists := false
    var existingUserIndex int

    for i, user := range usersData.Users {
        if user.UserID == userID {
            userExists = true
            existingUserIndex = i
            break
        }
    }

    if userExists {
        usersData.Users[existingUserIndex].PhoneNumber = phoneNumber
    } else {
        newUser := User{
            UserID:          userID,
            UserFirstName:   update.Message.From.FirstName,
            UserLastName:    update.Message.From.LastName,
            Username:        update.Message.From.UserName,
            PhoneNumber:     phoneNumber,
            RegistrationDate: time.Now(),
        }
        usersData.Users = append(usersData.Users, newUser)
    }

    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Номер телефона сохранен!")
    msg.ReplyToMessageID = update.Message.MessageID
    bot.Send(msg)
}

// Добавление нового пользователя, если его нет
func addUserIfNotExists(update tgbotapi.Update, usersData *UsersData) {
    userID := update.Message.From.ID
    userExists := false

    for _, user := range usersData.Users {
        if user.UserID == userID {
            userExists = true
            break
        }
    }

    if !userExists {
        newUser := User{
            UserID:          userID,
            UserFirstName:   update.Message.From.FirstName,
            UserLastName:    update.Message.From.LastName,
            Username:        update.Message.From.UserName,
            RegistrationDate: time.Now(),
        }
        usersData.Users = append(usersData.Users, newUser)
    }
}

// Обработка сообщения
func handleMessage(update tgbotapi.Update, messagesData *MessagesData, botToken string) Message {
    newMessage := Message{
        UserID:      update.Message.From.ID,
        MessageID:   update.Message.MessageID,
        Text:        update.Message.Text,
        MessageDate: time.Now(),
    }

    if update.Message.Photo != nil && len(update.Message.Photo) > 0 {
        var photoIDs []string
        var photoPaths []string

        photo := update.Message.Photo[len(update.Message.Photo)-1]
        photoIDs = append(photoIDs, photo.FileID)

        photoPath, err := saveMessagePhoto(botToken, photo.FileID)
        if err != nil {
            log.Printf("Ошибка при сохранении фото: %v", err)
        } else {
            photoPaths = append(photoPaths, photoPath)
        }

        newMessage.PhotoIDs = photoIDs
        newMessage.PhotoPaths = photoPaths
    }

    messagesData.Messages = append(messagesData.Messages, newMessage)
    return newMessage
}

// Формирование ответа
func createReplyText(update tgbotapi.Update, newMessage Message, avatarInfo string, err error) string {
    replyText := "Получены следующие данные:\n\n"
    replyText += "Текст сообщения: " + update.Message.Text + "\n"
    replyText += "ID сообщения: " + strconv.Itoa(update.Message.MessageID) + "\n"
    replyText += "Имя пользователя: " + update.Message.From.FirstName + "\n"
    replyText += "Фамилия пользователя: " + update.Message.From.LastName + "\n"
    replyText += "Никнейм пользователя: " + update.Message.From.UserName + "\n"
    replyText += "ID пользователя: " + strconv.FormatInt(update.Message.From.ID, 10) + "\n"

    if len(newMessage.PhotoIDs) > 0 {
        replyText += "К сообщению прикреплено фото\n"
    }

    if err != nil {
        replyText += "\nИнформация об аватаре: " + err.Error()
    } else {
        replyText += "\nИнформация об аватаре: " + avatarInfo
    }

    return replyText
}

func main() {
    botToken := "7217078454:AAGqrgEr_JuoJnwqwf1xU5P3lO--GnDtCIg"
    bot, err := tgbotapi.NewBotAPI(botToken)
    if err != nil {
        log.Panic(err)
    }

    bot.Debug = false
    log.Printf("Подключился как %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 10

    updates := bot.GetUpdatesChan(u)

    // Файлы для хранения данных
    usersDataFile := "users_data.json"
    messagesDataFile := "messages_data.json"

    for update := range updates {
        if update.Message == nil {
            continue
        }

        // Загрузка данных
        usersData, _ := loadUsersData(usersDataFile)
        messagesData, _ := loadMessagesData(messagesDataFile)

        // Обработка пользователя
        userExists := handleUser(update, &usersData)
        if !userExists {
            addUserIfNotExists(update, &usersData)
        }

        // Обработка контакта
        if update.Message.Contact != nil {
            handleContact(update, &usersData, bot)
        }

        // Обработка сообщения
        newMessage := handleMessage(update, &messagesData, botToken)

        // Сохранение данных
        saveUsersData(usersDataFile, usersData)
        saveMessagesData(messagesDataFile, messagesData)

        // Информация об аватаре
        avatarInfo, err := getUserAvatar(botToken, strconv.FormatInt(update.Message.From.ID, 10))

        // Формирование ответа
        replyText := createReplyText(update, newMessage, avatarInfo, err)

        // Отправка ответа
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
        msg.ReplyToMessageID = update.Message.MessageID
        bot.Send(msg)
    }
}