package bot

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"tgbot/config"
	"tgbot/models"
	"tgbot/state"
	"tgbot/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var tempBarbers = struct {
	sync.RWMutex
	M map[int64]models.Barber
}{M: make(map[int64]models.Barber)}

func HandleAdminStatistics(chatID int64, botInstance *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Statistika: (bu yerda statistikani chiqaring)")
	botInstance.Send(msg)
}

func HandleAdminAddTest(chatID int64, botInstance *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Barber qo'shish: Iltimos, barberning ID raqamini yuboring.")
	botInstance.Send(msg)

	state.UserStates.Lock()
	state.UserStates.M[chatID] = "adding_barber_id"
	state.UserStates.Unlock()
}

func HandleAdminDeleteTest(chatID int64, botInstance *tgbotapi.BotAPI) {
	barbers, err := storage.GetBarbers()
	if err != nil {
		log.Printf("Sartaroshlarni olishda xatolik: %v", err)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(barbers); i += 2 {
		if i+1 < len(barbers) {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(barbers[i].Name, fmt.Sprintf("delete_%s", barbers[i].Name)),
				tgbotapi.NewInlineKeyboardButtonData(barbers[i+1].Name, fmt.Sprintf("delete_%s", barbers[i+1].Name)),
			))
		} else {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(barbers[i].Name, fmt.Sprintf("delete_%s", barbers[i].Name)),
			))
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, "Sartaroshni tanlang:")
	msg.ReplyMarkup = keyboard

	if _, err := botInstance.Send(msg); err != nil {
		log.Printf("Sartaroshlarni tanlash tugmalarini yuborishda xatolik: %v", err)
	}

	state.UserStates.Lock()
	state.UserStates.M[chatID] = "selecting_barber_for_deletion"
	state.UserStates.Unlock()
}

func HandleDeleteBarberCallback(chatID int64, barberName string, botInstance *tgbotapi.BotAPI, prevMessageID int) {

	deleteMessage := tgbotapi.NewDeleteMessage(chatID, prevMessageID)
	botInstance.Send(deleteMessage)

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Siz rostdan ham %s ni o'chirishni istaysizmi?", barberName))
	yesButton := tgbotapi.NewInlineKeyboardButtonData("Ha", fmt.Sprintf("confirm_delete_%s", barberName))
	noButton := tgbotapi.NewInlineKeyboardButtonData("Yo'q", "cancel_delete")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(yesButton, noButton),
	)
	msg.ReplyMarkup = keyboard

	if _, err := botInstance.Send(msg); err != nil {
		log.Printf("Tasdiqlash tugmalarini yuborishda xatolik: %v", err)
	}
}

func HandleDeleteBarberConfirmation(chatID int64, barberName string, botInstance *tgbotapi.BotAPI, prevMessageID int) {

	deleteMessage := tgbotapi.NewDeleteMessage(chatID, prevMessageID)
	botInstance.Send(deleteMessage)

	err := storage.DeleteBarber(barberName)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Barberni o'chirishda xatolik: %v", err))
		botInstance.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Barber %s muvaffaqiyatli o'chirildi.", barberName))
	botInstance.Send(msg)
}

func HandleAddBarberID(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text

	id, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		reply := tgbotapi.NewMessage(chatID, "Noto'g'ri ID raqam, iltimos qayta yuboring:")
		config.GetBot().Send(reply)
		return
	}

	tempBarbers.Lock()
	tempBarbers.M[chatID] = models.Barber{ID: id}
	tempBarbers.Unlock()

	reply := tgbotapi.NewMessage(chatID, "Barberning ismini yuboring:")
	config.GetBot().Send(reply)

	state.UserStates.Lock()
	state.UserStates.M[chatID] = "adding_barber_name"
	state.UserStates.Unlock()
}

func HandleAddBarberName(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text

	tempBarbers.Lock()
	barber := tempBarbers.M[chatID]
	barber.Name = text
	tempBarbers.M[chatID] = barber
	tempBarbers.Unlock()

	reply := tgbotapi.NewMessage(chatID, "Barberning usernamesini yuboring:")
	config.GetBot().Send(reply)

	state.UserStates.Lock()
	state.UserStates.M[chatID] = "adding_barber_username"
	state.UserStates.Unlock()
}

func HandleAddBarberUserName(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text

	tempBarbers.Lock()
	barber := tempBarbers.M[chatID]
	barber.UserName = text
	tempBarbers.M[chatID] = barber
	tempBarbers.Unlock()

	reply := tgbotapi.NewMessage(chatID, "Barberning telefon raqamini yuboring:")
	config.GetBot().Send(reply)

	state.UserStates.Lock()
	state.UserStates.M[chatID] = "adding_barber_phone"
	state.UserStates.Unlock()
}

func HandleAddBarberPhone(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text

	tempBarbers.Lock()
	barber := tempBarbers.M[chatID]
	barber.Phone = text
	tempBarbers.Unlock()

	err := storage.AddBarber(barber)
	if err != nil {
		reply := tgbotapi.NewMessage(chatID, "Barberni qo'shishda xatolik: " + err.Error())
		config.GetBot().Send(reply)
	} else {
		reply := tgbotapi.NewMessage(chatID, "Barber muvaffaqiyatli qo'shildi.")
		config.GetBot().Send(reply)
	}

	state.UserStates.Lock()
	delete(state.UserStates.M, chatID)
	state.UserStates.Unlock()

	tempBarbers.Lock()
	delete(tempBarbers.M, chatID)
	tempBarbers.Unlock()
}
