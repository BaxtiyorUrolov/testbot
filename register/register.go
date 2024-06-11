package register

import (
	"database/sql"
	"fmt"
	"log"

	"tgbot/stats"
	"tgbot/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)


func HandleFullName(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	fmt.Println("Ism: ", text)

	err := storage.UpdateUserFullName(db, chatID, text)
	if err != nil {
		log.Printf("Error updating full name: %v", err)
		return
	}
	stats.UsStats[chatID] = "waiting_for_region"
	msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, viloyatingizni kiriting:")
	botInstance.Send(msgResponse)
}

func HandleRegion(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	storage.UpdateUserRegion(db, chatID, text)
	stats.UsStats[chatID] = "waiting_for_district"
	msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, tumaningizni kiriting:")
	botInstance.Send(msgResponse)
}

func HandleDistrict(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	storage.UpdateUserDistrict(db, chatID, text)
	stats.UsStats[chatID] = "waiting_for_school"
	msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, maktabingizni kiriting: \n\n Namuna: 68")
	botInstance.Send(msgResponse)
}

func HandleSchool(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	storage.UpdateUserSchool(db, chatID, text)
	stats.UsStats[chatID] = "waiting_for_grade"
	msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, sinfingizni kiriting:\n\n Namuna: 10")
	botInstance.Send(msgResponse)
}

func HandleGrade(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	text := msg.Text

	storage.UpdateUserGrade(db, chatID, text)
	stats.UsStats[chatID] = "waiting_for_phone"

	// Create custom keyboard with the "Share Phone Number" button
	sharePhoneButton := tgbotapi.NewKeyboardButtonContact("Telefon raqamni ulashish")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(sharePhoneButton),
	)
	msgResponse := tgbotapi.NewMessage(chatID, "Iltimos, telefon raqamingizni ulashing:")
	msgResponse.ReplyMarkup = keyboard
	botInstance.Send(msgResponse)
}


func HandlePhone(msg *tgbotapi.Message, db *sql.DB, botInstance *tgbotapi.BotAPI) {
	chatID := msg.Chat.ID
	var phoneNumber string

	if msg.Contact != nil {
		phoneNumber = msg.Contact.PhoneNumber
	} else {
		phoneNumber = msg.Text
	}

	storage.UpdateUserPhone(db, chatID, phoneNumber)
	delete(stats.UsStats, chatID)
	
	// Remove the custom keyboard
	removeKeyboard := tgbotapi.NewRemoveKeyboard(true)
	msgResponse := tgbotapi.NewMessage(chatID, "Ro'yxatdan o'tish muvaffaqiyatli yakunlandi!")
	msgResponse.ReplyMarkup = removeKeyboard

	// Send the message about successful registration and remove the keyboard
	botInstance.Send(msgResponse)
	
	// Now send the message with the inline button to start the test
	startTestButton := tgbotapi.NewInlineKeyboardButtonData("Testni boshlash", "start_test")
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(startTestButton),
	)
	testMsg := tgbotapi.NewMessage(chatID, "Testni boshlash uchun quyidagi tugmani bosing.")
	testMsg.ReplyMarkup = inlineKeyboard
	botInstance.Send(testMsg)
}


