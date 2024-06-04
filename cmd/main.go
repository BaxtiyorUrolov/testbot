package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"tgbot/storage"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var userStates = make(map[int64]string)

func main() {
	connStr := "user=godb password=0208 dbname=testbot sslmode=disable"
	db, err := storage.OpenDatabase(connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	botToken := "5111237025:AAHhUYhFG4xuu6hVjhka8YuBYNBVnrtzGps"
	botInstance, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	offset := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down bot...")
			return
		default:
			updates, err := botInstance.GetUpdates(tgbotapi.NewUpdate(offset))
			if err != nil {
				log.Printf("Error getting updates: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			for _, update := range updates {
				handleUpdate(update, db, botInstance)
				offset = update.UpdateID + 1
			}
		}
	}
}

func handleUpdate(update tgbotapi.Update, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	if update.Message != nil {
		handleMessage(update.Message, db, botInstance)
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(update.CallbackQuery, db, botInstance)
	} else {
		log.Printf("Unsupported update type: %T", update)
	}
}

func handleMessage(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	log.Printf("Received message: %s", text)

	if state, exists := userStates[chatID]; exists {
		switch state {
		case "waiting_for_channel_link":
			handleChannelLink(msg, db, botInstance)
			delete(userStates, chatID)
			return
		case "waiting_for_answers":
			handleAnswers(msg, db, botInstance)
			delete(userStates, chatID)
			return
		case "waiting_for_test_file":
			handleDocument(msg, db, botInstance)
			delete(userStates, chatID)
			return
		case "waiting_for_test_answers":
			handleTestAnswers(msg, db, botInstance)
			delete(userStates, chatID)
			return
		case "waiting_for_admin_id":
			handleAdminAdd(msg, db, botInstance)
			delete(userStates, chatID)
			return
		case "waiting_for_admin_id_remove":
			handleAdminRemove(msg, db, botInstance)
			delete(userStates, chatID)
			return
		}
	}

	if text == "/start" {
		handleStartCommand(msg, db, botInstance)
	} else if text == "/admin" {
		handleAdminCommand(msg, db, botInstance)
	} else {
		handleDefaultMessage(msg, db, botInstance)
	}
}

func handleStartCommand(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	userName := msg.From.UserName

	log.Printf("Adding user to database: %d - %s", userID, userName)
	err := storage.AddUserToDatabase(db, userID, userName)
	if err != nil {
		log.Printf("Error adding user to database: %v", err)
		return
	}

	channels, err := storage.GetChannelsFromDatabase(db)
	if err != nil {
		log.Printf("Error getting channels from database: %v", err)
		return
	}

	log.Printf("Checking subscription for user %d", chatID)
	if isUserSubscribedToChannels(chatID, channels, botInstance) {
		msg := tgbotapi.NewMessage(chatID, "Assalomu alaykum, botimizga xush kelibsiz!")
		startTestButton := tgbotapi.NewInlineKeyboardButtonData("Testni boshlash", "start_test")
		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(startTestButton),
		)
		msg.ReplyMarkup = inlineKeyboard
		botInstance.Send(msg)
	} else {
		log.Printf("User %d is not subscribed to required channels", chatID)
		inlineKeyboard := createSubscriptionKeyboard(channels)
		msg := tgbotapi.NewMessage(chatID, "Iltimos, avval kanallarga azo bo'ling:")
		msg.ReplyMarkup = inlineKeyboard
		botInstance.Send(msg)
	}
}

func handleAdminCommand(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID

	if !storage.IsAdmin(int(chatID), db) {
		msgResponse := tgbotapi.NewMessage(chatID, "Siz admin emassiz.")
		botInstance.Send(msgResponse)
		return
	}

	adminKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Statistika"),
			tgbotapi.NewKeyboardButton("Kanal qo'shish"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Kanal o'chirish"),
			tgbotapi.NewKeyboardButton("Test faylini yuklash"),
			tgbotapi.NewKeyboardButton("Test javoblarini yuklash"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Admin qo'shish"),
			tgbotapi.NewKeyboardButton("Admin o'chirish"),
		),
	)

	msgResponse := tgbotapi.NewMessage(chatID, "Admin buyrug'lari:")
	msgResponse.ReplyMarkup = adminKeyboard
	botInstance.Send(msgResponse)
}

func handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID

	channels, err := storage.GetChannelsFromDatabase(db)
	if err != nil {
		log.Printf("Error getting channels from database: %v", err)
		return
	}

	if callbackQuery.Data == "check_subscription" {
		if isUserSubscribedToChannels(chatID, channels, botInstance) {
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
			botInstance.Send(deleteMsg)
			msg := tgbotapi.NewMessage(chatID, "Assalomu alaykum, siz kanallarga azo bo'ldingiz!")
			startTestButton := tgbotapi.NewInlineKeyboardButtonData("Testni boshlash", "start_test")
			inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(startTestButton),
			)
			msg.ReplyMarkup = inlineKeyboard
			botInstance.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(chatID, "Iltimos, kanallarga azo bo'ling.")
			inlineKeyboard := createSubscriptionKeyboard(channels)
			msg.ReplyMarkup = inlineKeyboard
			botInstance.Send(msg)
		}
	} else if callbackQuery.Data == "start_test" {
		handleStartTest(chatID, messageID, db, botInstance)
	} else if callbackQuery.Data == "check_answers" {
		handleCheckAnswers(chatID, messageID, botInstance)
	} else if strings.HasPrefix(callbackQuery.Data, "delete_channel_") {
		channel := strings.TrimPrefix(callbackQuery.Data, "delete_channel_")
		askForChannelDeletionConfirmation(chatID, messageID, channel, botInstance)
	} else if strings.HasPrefix(callbackQuery.Data, "confirm_delete_channel_") {
		channel := strings.TrimPrefix(callbackQuery.Data, "confirm_delete_channel_")
		deleteChannel(chatID, messageID, channel, db, botInstance)
	} else if callbackQuery.Data == "cancel_delete_channel" {
		cancelChannelDeletion(chatID, messageID, botInstance)
	}
}

func handleStartTest(chatID int64, messageID int, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	// Delete the previous message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	botInstance.Send(deleteMsg)

	fileID, fileName, err := storage.GetFileFromDatabase(db)
	if err != nil {
		log.Printf("Error getting file from database: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Faylni olishda xatolik yuz berdi.")
		botInstance.Send(msg)
		return
	}

	fileBytes, err := downloadFile(botInstance, fileID)
	if err != nil {
		log.Printf("Error downloading file: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Faylni olishda xatolik yuz berdi.")
		botInstance.Send(msg)
		return
	}

	document := tgbotapi.NewDocumentUpload(chatID, tgbotapi.FileBytes{
		Name:  fileName,
		Bytes: fileBytes,
	})
	if _, err := botInstance.Send(document); err != nil {
		log.Printf("Error sending document: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Faylni yuborishda xatolik yuz berdi.")
		botInstance.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Test faylini oling. Javoblaringizni tekshirish uchun quyidagi tugmani bosing.")
	checkAnswersButton := tgbotapi.NewInlineKeyboardButtonData("Javoblarni tekshirish", "check_answers")
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(checkAnswersButton),
	)
	msg.ReplyMarkup = inlineKeyboard
	botInstance.Send(msg)
}

func handleCheckAnswers(chatID int64, messageID int, botInstance *tgbotapi.BotAPI) {
	// Delete the previous message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	botInstance.Send(deleteMsg)

	userStates[chatID] = "waiting_for_answers"
	msg := tgbotapi.NewMessage(chatID, "Iltimos, javoblaringizni yuboring:")
	botInstance.Send(msg)
}

func handleDefaultMessage(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	switch text {
	case "Kanal qo'shish":
		userStates[chatID] = "waiting_for_channel_link"
		msgResponse := tgbotapi.NewMessage(chatID, "Kanal linkini yuboring (masalan, https://t.me/your_channel):")
		botInstance.Send(msgResponse)
	case "Test faylini yuklash":
		userStates[chatID] = "waiting_for_test_file"
		msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, test faylini yuklang:")
		botInstance.Send(msgResponse)
	case "Test javoblarini yuklash":
		userStates[chatID] = "waiting_for_test_answers"
		msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, test javoblarini yuboring:")
		botInstance.Send(msgResponse)
	case "Admin qo'shish":
		userStates[chatID] = "waiting_for_admin_id"
		msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, yangi admin ID sini yuboring:")
		botInstance.Send(msgResponse)
	case "Admin o'chirish":
		userStates[chatID] = "waiting_for_admin_id_remove"
		msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, admin ID sini o'chirish uchun yuboring:")
		botInstance.Send(msgResponse)
	case "Kanal o'chirish":
		displayChannelsForDeletion(chatID, db, botInstance)
	default:
		msgResponse := tgbotapi.NewMessage(chatID, "Har qanday boshqa xabarlarni shu yerda ko'rib chiqish mumkin")
		botInstance.Send(msgResponse)
	}
}

func handleChannelLink(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	channelLink := msg.Text

	err := storage.AddChannelToDatabase(db, channelLink)
	if err != nil {
		log.Printf("Error adding channel to database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Kanalni qo'shishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, "Kanal muvaffaqiyatli qo'shildi.")
	botInstance.Send(msgResponse)
}

func handleDocument(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	fileID := msg.Document.FileID
	fileName := msg.Document.FileName
	mimeType := msg.Document.MimeType

	log.Printf("Received document: %s", fileName)
	err := saveFile(db, botInstance, fileID, fileName, mimeType)
	if err != nil {
		log.Printf("Error saving file: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Faylni saqlashda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	userStates[chatID] = "waiting_for_answers"
	msgResponse := tgbotapi.NewMessage(chatID, "Fayl muvaffaqiyatli saqlandi. Iltimos, endi javoblarni yuboring:")
	botInstance.Send(msgResponse)
}

func handleTestAnswers(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID

	fmt.Println(msg.Text)

	err := storage.AddAnswerToDatabase(db, msg.Text)
	if err != nil {
		log.Printf("Error adding answer to database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Javoblarni qo'shishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, "Javoblar muvaffaqiyatli qo'shildi.")
	botInstance.Send(msgResponse)
}

func handleAnswers(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	userAnswers := msg.Text

	log.Printf("Received answers: %s", userAnswers)

	correctAnswers, err := storage.GetCorrectAnswersFromDatabase(db)
	if err != nil {
		log.Printf("Error getting correct answers: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Javoblarni tekshirishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	log.Printf("Correct answers: %s", correctAnswers)

	err = storage.AddAnswerToDatabase(db, userAnswers)
	if err != nil {
		log.Printf("Error saving answers: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Javoblarni saqlashda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	correctCount, incorrectIndices := checkAnswers(userAnswers, correctAnswers)

	msgResponse := tgbotapi.NewMessage(chatID, fmt.Sprintf("Javoblaringiz tekshirildi. To'g'ri javoblar soni: %d", correctCount))
	if len(incorrectIndices) > 0 {
		msgResponse.Text += fmt.Sprintf("\nNoto'g'ri javoblar: %s", strings.Join(intSliceToStringSlice(incorrectIndices), ", "))
	}
	botInstance.Send(msgResponse)
}

func handleAdminAdd(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	adminID, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil {
		log.Printf("Error parsing admin ID: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Noto'g'ri admin ID formati.")
		botInstance.Send(msgResponse)
		return
	}

	err = storage.AddAdminToDatabase(db, adminID)
	if err != nil {
		log.Printf("Error adding admin to database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Admin qo'shishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, "Admin muvaffaqiyatli qo'shildi.")
	botInstance.Send(msgResponse)
}

func handleAdminRemove(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	adminID, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil {
		log.Printf("Error parsing admin ID: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Noto'g'ri admin ID formati.")
		botInstance.Send(msgResponse)
		return
	}

	err = storage.RemoveAdminFromDatabase(db, adminID)
	if err != nil {
		log.Printf("Error removing admin from database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Admin o'chirishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, "Admin muvaffaqiyatli o'chirildi.")
	botInstance.Send(msgResponse)
}

func displayChannelsForDeletion(chatID int64, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	channels, err := storage.GetChannelsFromDatabase(db)
	if err != nil {
		log.Printf("Error getting channels from database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Kanallarni olishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, channel := range channels {
		button := tgbotapi.NewInlineKeyboardButtonData(channel, "delete_channel_"+channel)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msgResponse := tgbotapi.NewMessage(chatID, "O'chirilishi kerak bo'lgan kanalni tanlang:")
	msgResponse.ReplyMarkup = inlineKeyboard
	botInstance.Send(msgResponse)
}

func askForChannelDeletionConfirmation(chatID int64, messageID int, channel string, botInstance *tgbotapi.BotAPI) {
	confirmButton := tgbotapi.NewInlineKeyboardButtonData("Ha", "confirm_delete_channel_"+channel)
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("Yo'q", "cancel_delete_channel")

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(confirmButton, cancelButton),
	)
	msgResponse := tgbotapi.NewMessage(chatID, fmt.Sprintf("%s kanalini o'chirmoqchimisiz?", channel))
	msgResponse.ReplyMarkup = inlineKeyboard
	botInstance.Send(msgResponse)

	// Delete the previous message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	botInstance.Send(deleteMsg)
}

func deleteChannel(chatID int64, messageID int, channel string, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	err := storage.DeleteChannelFromDatabase(db, channel)
	if err != nil {
		log.Printf("Error deleting channel from database: %v", err)
		msgResponse := tgbotapi.NewMessage(chatID, "Kanalni o'chirishda xatolik yuz berdi.")
		botInstance.Send(msgResponse)
		return
	}

	msgResponse := tgbotapi.NewMessage(chatID, fmt.Sprintf("%s kanali muvaffaqiyatli o'chirildi.", channel))
	botInstance.Send(msgResponse)
}

func cancelChannelDeletion(chatID int64, messageID int, botInstance *tgbotapi.BotAPI) {
	msgResponse := tgbotapi.NewMessage(chatID, "Kanal o'chirish bekor qilindi.")
	botInstance.Send(msgResponse)

	// Delete the previous message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	botInstance.Send(deleteMsg)
}

func checkAnswers(userAnswers, correctAnswers string) (int, []int) {
	userAns := strings.ReplaceAll(userAnswers, "\n", "")
	correctAns := strings.ReplaceAll(correctAnswers, "\n", "")

	count := 0
	var incorrectIndices []int
	for i := 0; i < len(userAns) && i < len(correctAns); i++ {
		if userAns[i] == correctAns[i] {
			count++
		} else {
			incorrectIndices = append(incorrectIndices, i+1) // Indices are 1-based for user readability
		}
	}

	return count, incorrectIndices
}

func saveFile(db *sql.DB, botInstance *tgbotapi.BotAPI, fileID, fileName, mimeType string) error {
	fileConfig, err := botInstance.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return fmt.Errorf("error getting file config: %v", err)
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botInstance.Token, fileConfig.FilePath)

	response, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer response.Body.Close()

	fileData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading file data: %v", err)
	}

	err = storage.AddFileMetadataToDatabase(db, fileID, fileName, mimeType, fileData)
	if err != nil {
		return fmt.Errorf("error saving file metadata to database: %v", err)
	}

	return nil
}

func saveAnswers(db *sql.DB, botInstance *tgbotapi.BotAPI, fileID, fileName, mimeType string) error {
	fileConfig, err := botInstance.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return fmt.Errorf("error getting file config: %v", err)
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botInstance.Token, fileConfig.FilePath)

	response, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer response.Body.Close()

	fileData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading file data: %v", err)
	}

	err = storage.TruncateAnswersTable(db)
	if err != nil {
		return fmt.Errorf("error truncating answers table: %v", err)
	}

	err = storage.AddAnswersToDatabase(db, fileData)
	if err != nil {
		return fmt.Errorf("error saving answers to database: %v", err)
	}

	return nil
}

func downloadFile(botInstance *tgbotapi.BotAPI, fileID string) ([]byte, error) {
	fileConfig, err := botInstance.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("error getting file config: %v", err)
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botInstance.Token, fileConfig.FilePath)
	response, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("error downloading file: %v", err)
	}
	defer response.Body.Close()

	fileBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading file data: %v", err)
	}

	return fileBytes, nil
}

func isUserSubscribedToChannels(chatID int64, channels []string, botInstance *tgbotapi.BotAPI) bool {
	for _, channel := range channels {
		log.Printf("Checking subscription to channel: %s", channel)
		chat, err := botInstance.GetChat(tgbotapi.ChatConfig{SuperGroupUsername: "@" + channel})
		if err != nil {
			log.Printf("Error getting chat info for channel %s: %v", channel, err)
			return false
		}

		member, err := botInstance.GetChatMember(tgbotapi.ChatConfigWithUser{
			ChatID: chat.ID,
			UserID: int(chatID),
		})
		if err != nil {
			log.Printf("Error getting chat member info for channel %s: %v", channel, err)
			return false
		}
		if member.Status == "left" || member.Status == "kicked" {
			log.Printf("User %d is not subscribed to channel %s", chatID, channel)
			return false
		}
	}
	return true
}

func createSubscriptionKeyboard(channels []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, channel := range channels {
		channelName := strings.TrimPrefix(channel, "https://t.me/")
		button := tgbotapi.NewInlineKeyboardButtonURL(channelName, "https://t.me/"+channelName)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}
	checkButton := tgbotapi.NewInlineKeyboardButtonData("Azo bo'ldim", "check_subscription")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(checkButton))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func intSliceToStringSlice(intSlice []int) []string {
	var stringSlice []string
	for _, val := range intSlice {
		stringSlice = append(stringSlice, fmt.Sprint(val))
	}
	return stringSlice
}
