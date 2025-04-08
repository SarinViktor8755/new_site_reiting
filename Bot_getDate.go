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

// Структура для хранения информации о пользователях в JSON
type User struct {
    UserID          int64     `json:"userID"`
    UserFirstName   string    `json:"userFirstName"`
    UserLastName    string    `json:"userLastName,omitempty"` // Новое поле для фамилии
    Username        string    `json:"username,omitempty"`     // Новое поле для никнейма
    PhoneNumber     string    `json:"phoneNumber,omitempty"`  // Номер телефона
    RegistrationDate time.Time `json:"registrationDate"`
}

type UsersData struct {
    Users []User `json:"users"`
}

// Функция для загрузки данных из JSON файла
func loadUsersData(filename string) (UsersData, error) {
    var usersData UsersData

    // Проверяем, существует ли файл
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        // Если файла нет, создаем пустую структуру
        return UsersData{Users: []User{}}, nil
    }

    // Читаем данные из файла
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

// Функция для сохранения данных в JSON файл
func saveUsersData(filename string, usersData UsersData) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ") // Для красивого форматирования JSON
    err = encoder.Encode(usersData)
    if err != nil {
        return err
    }

    return nil
}

// getUserAvatar получает и сохраняет аватар пользователя
func getUserAvatar(botToken string, userID string) (string, error) {
    // Шаг 1: Получить фото профиля пользователя
    photosURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUserProfilePhotos?user_id=%s", botToken, userID)
    resp, err := http.Get(photosURL)
    if err != nil {
        return "", fmt.Errorf("ошибка при запросе к API: %v", err)
    }
    defer resp.Body.Close()

    // Декодируем JSON-ответ
    var userProfilePhotos UserProfilePhotos
    err = json.NewDecoder(resp.Body).Decode(&userProfilePhotos)
    if err != nil {
        return "", fmt.Errorf("ошибка при декодировании JSON: %v", err)
    }

    // Проверяем, есть ли фотографии
    if userProfilePhotos.Result.TotalCount == 0 {
        return "у пользователя нет аватара", nil
    }

    // Берем первый файл ID из списка
    fileID := userProfilePhotos.Result.Photos[0][0].FileID

    // Шаг 2: Получить путь к файлу
    fileInfoURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", botToken, fileID)
    resp, err = http.Get(fileInfoURL)
    if err != nil {
        return "", fmt.Errorf("ошибка при запросе к getFile: %v", err)
    }
    defer resp.Body.Close()

    // Декодируем JSON-ответ
    var fileResponse FileResponse
    err = json.NewDecoder(resp.Body).Decode(&fileResponse)
    if err != nil {
        return "", fmt.Errorf("ошибка при декодировании JSON для getFile: %v", err)
    }

    // Формируем URL для скачивания файла
    fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, fileResponse.Result.FilePath)

    // Шаг 3: Скачиваем файл
    resp, err = http.Get(fileURL)
    if err != nil {
        return "", fmt.Errorf("ошибка при скачивании файла: %v", err)
    }
    defer resp.Body.Close()

    // Сохраняем файл локально
    filename := userID + ".jpg"
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

func main() {
    botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
    if botToken == "" {
        log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
    }

    bot, err := tgbotapi.NewBotAPI("7217078454:AAGqrgEr_JuoJnwqwf1xU5P3lO--GnDtCIg")
    if err != nil {
        log.Panic(err)
    }

    bot.Debug = true
    log.Printf("Authorized on account %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 10

    updates := bot.GetUpdatesChan(u)

    // Имя файла для хранения данных о пользователях
    usersDataFile := "users_data.json"

    for update := range updates {
        if update.Message == nil {
            continue
        }

        // Загружаем данные о пользователях из файла
        usersData, err := loadUsersData(usersDataFile)
        if err != nil {
            log.Printf("Ошибка при загрузке данных о пользователях: %v", err)
        }

        // Извлекаем данные
        userID := update.Message.From.ID
        userFirstName := update.Message.From.FirstName
        userLastName := update.Message.From.LastName // Фамилия
        username := update.Message.From.UserName    // Никнейм

        // Проверяем, есть ли уже пользователь в файле
        userExists := false
        var existingUserIndex int
        for i, user := range usersData.Users {
            if user.UserID == userID {
                userExists = true
                existingUserIndex = i
                break
            }
        }

        // Если пользователь отправил контакт
        if update.Message.Contact != nil {
            phoneNumber := update.Message.Contact.PhoneNumber

            // Если пользователь уже существует, обновляем номер телефона
            if userExists {
                usersData.Users[existingUserIndex].PhoneNumber = phoneNumber
            } else {
                // Если пользователя нет, добавляем нового
                newUser := User{
                    UserID:          userID,
                    UserFirstName:   userFirstName,
                    UserLastName:    userLastName, // Сохраняем фамилию
                    Username:        username,    // Сохраняем никнейм
                    PhoneNumber:     phoneNumber,
                    RegistrationDate: time.Now(),
                }
                usersData.Users = append(usersData.Users, newUser)
            }

            // Сохраняем обновленные данные в файл
            err = saveUsersData(usersDataFile, usersData)
            if err != nil {
                log.Printf("Ошибка при сохранении данных о пользователях: %v", err)
            }

            // Отправляем подтверждение
            msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Номер телефона сохранен!")
            msg.ReplyToMessageID = update.Message.MessageID
            if _, err := bot.Send(msg); err != nil {
                log.Panic(err)
            }
            continue
        }

        // Если пользователь отправил текстовое сообщение
        messageText := update.Message.Text
        messageID := update.Message.MessageID

        // Если пользователя нет, добавляем его
        if !userExists {
            newUser := User{
                UserID:          userID,
                UserFirstName:   userFirstName,
                UserLastName:    userLastName, // Сохраняем фамилию
                Username:        username,    // Сохраняем никнейм
                RegistrationDate: time.Now(),
            }
            usersData.Users = append(usersData.Users, newUser)

            // Сохраняем обновленные данные в файл
            err = saveUsersData(usersDataFile, usersData)
            if err != nil {
                log.Printf("Ошибка при сохранении данных о пользователях: %v", err)
            }
        }

        // Формируем ответ
        replyText := "Получены следующие данные:\n\n"
        replyText += "Текст сообщения: " + messageText + "\n"
        replyText += "ID сообщения: " + strconv.Itoa(messageID) + "\n"
        replyText += "Имя пользователя: " + userFirstName + "\n"
        replyText += "Фамилия пользователя: " + userLastName + "\n" // Выводим фамилию
        replyText += "Никнейм пользователя: " + username + "\n"    // Выводим никнейм
        replyText += "ID пользователя: " + strconv.FormatInt(userID, 10) + "\n"

        if userExists {
            replyText += "\nПользователь уже существует в базе данных."
        } else {
            replyText += "\nПользователь успешно добавлен в базу данных."
        }

        // Добавляем информацию об аватаре
        avatarInfo, err := getUserAvatar(botToken, strconv.FormatInt(userID, 10))
        if err != nil {
            replyText += "\nИнформация об аватаре: " + err.Error()
        } else {
            replyText += "\nИнформация об аватаре: " + avatarInfo
        }

        msg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
        msg.ReplyToMessageID = update.Message.MessageID

        if _, err := bot.Send(msg); err != nil {
            log.Panic(err)
        }
    }
}