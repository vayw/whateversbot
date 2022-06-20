package main

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

var NumericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("count", "count"),
		tgbotapi.NewInlineKeyboardButtonData("ticket", "ticket"),
	),
)

func BotCallback(query *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	switch query.Data {
	case "ticket":
		new_id := uuid.New()
		url := fmt.Sprintf("https://jzzy.ru/ticket/%v", new_id)
		png, _ := qrcode.Encode(url, qrcode.Medium, 256)
		photoFileBytes := tgbotapi.FileBytes{
			Name:  "ticket",
			Bytes: png,
		}
		media := tgbotapi.NewPhoto(query.Message.Chat.ID, photoFileBytes)

		if _, err := bot.Send(media); err != nil {
			log.Println(err)
		}
	default:
		newmsg := tgbotapi.NewMessage(query.Message.Chat.ID, "")
		newmsg.Text = "здорова, мышь"
		if _, err := bot.Send(newmsg); err != nil {
			log.Println(err)
		}
	}
}

func BotAnswer(msg *tgbotapi.Message, bot *tgbotapi.BotAPI, conf *Config) {
	newmsg := tgbotapi.NewMessage(msg.Chat.ID, "")
	if msg.IsCommand() {
		switch msg.Command() {
		case "count":
			count, err := VKcli.MembersCount()
			if err != nil {
				newmsg.Text = "что-то не получилось."
			}
			newmsg.Text = fmt.Sprintf("количество участников в группе: %d", count)
			newmsg.ReplyToMessageID = msg.MessageID
		case "hello":
			newmsg.Text = "hello!"
			newmsg.ReplyMarkup = NumericKeyboard
		default:
			newmsg.ReplyMarkup = NumericKeyboard
		}
	} else {
		if isfriend(msg.From.ID, conf) {
			newmsg.Text = msg.Text
			newmsg.ReplyToMessageID = msg.MessageID
		} else {
			log.Printf("%d", msg.From.ID)
			newmsg.Text = "извините, но мы не друзья"
			newmsg.ReplyToMessageID = msg.MessageID
		}
	}
	if _, err := bot.Send(newmsg); err != nil {
		log.Println(err)
	}
}
