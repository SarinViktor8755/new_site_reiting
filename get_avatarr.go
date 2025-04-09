package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

// Структура для ответа Telegram API
type UserProfilePhotos struct {
    Ok     bool `json:"ok"`
    Result struct {
        TotalCount int `json:"total_count"`
        Photos     [][]struct {
            FileID string `json:"file_id"`
        } `json:"photos"`
    } `json:"result"`
}

// Структура для ответа getFile
type FileResponse struct {
    Ok     bool `json:"ok"`
    Result struct {
        FilePath string `json:"file_path"`
    } `json:"result"`
}

func main() {
	// Замените на ваш токен и ID пользователя
	botToken := "7217078454:AAGqrgEr_JuoJnwqwf1xU5P3lO--GnDtCIg"
	userID := "370195432"

	// Попытка создать файл блокировки
	lockFile, err := os.OpenFile("bot.lock", os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println("Не удалось создать файл блокировки:", err)
		return
	}
	defer lockFile.Close()

	// Попытка заблокировать файл
	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		fmt.Println("завершено другим запросом getUpdates; убедитесь, что запущен только один экземпляр бота")
		return
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
    photosURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUserProfilePhotos?user_id=%s", botToken, userID)
    resp, err := http.Get(photosURL)
	fmt.Println("Ошибка при запросе к getFile:", photosURL)
    if err != nil {
        fmt.Println("Ошибка при запросе к API:", err)
        return
    }
    defer resp.Body.Close()

    // Декодируем JSON-ответ
    var userProfilePhotos UserProfilePhotos
    err = json.NewDecoder(resp.Body).Decode(&userProfilePhotos)
    if err != nil {
        fmt.Println("Ошибка при декодировании JSON:", err)
        return
    }

    // Проверяем, есть ли фотографии
    if userProfilePhotos.Result.TotalCount == 0 {
        fmt.Println("У пользователя нет аватара.")
        return
    }

    // Берем первый файл ID из списка
    fileID := userProfilePhotos.Result.Photos[0][0].FileID

    // Шаг 2: Получить путь к файлу
    fileInfoURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", botToken, fileID)
    resp, err = http.Get(fileInfoURL)
    if err != nil {
        fmt.Println("Ошибка при запросе к getFile:", err)
        return
    }
    defer resp.Body.Close()

    // Декодируем JSON-ответ
    var fileResponse FileResponse
    err = json.NewDecoder(resp.Body).Decode(&fileResponse)
    if err != nil {
        fmt.Println("Ошибка при декодировании JSON для getFile:", err)
        return
    }

    // Формируем URL для скачивания файла
    fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, fileResponse.Result.FilePath)

    // Шаг 3: Скачиваем файл
    resp, err = http.Get(fileURL)
    if err != nil {
        fmt.Println("Ошибка при скачивании файла:", err)
        return
    }
    defer resp.Body.Close()

    // Сохраняем файл локально
    out, err := os.Create(userID+".jpg")
    if err != nil {
        fmt.Println("Ошибка при создании файла:", err)
        return
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    if err != nil {
        fmt.Println("Ошибка при записи файла:", err)
        return
    }

    fmt.Println("Аватар успешно скачан как "+userID+".jpg")
}