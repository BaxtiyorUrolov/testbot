package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func Admin(chatID int64, botInstance *tgbotapi.BotAPI) {
	// Create reply keyboard buttons
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Statistika"),
			tgbotapi.NewKeyboardButton("Barber qo'shish"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Barber o'chirish"),
		),
	)

	// Create message with the reply keyboard
	msg := tgbotapi.NewMessage(chatID, "Admin panel:")
	msg.ReplyMarkup = keyboard

	// Send the message
	_, err := botInstance.Send(msg)
	if err != nil {
		log.Printf("Error sending admin message: %v", err)
	}
}
