package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	vkapi "github.com/vayw/gosocial"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type Config struct {
	VK         VKconf
	TG         TGconf
	StatusFile string
	Save       int
}

type VKconf struct {
	APIkey  string
	GroupID string
}

type TGconf struct {
	API     string
	Friends []int
	Channel int64
}

type Status struct {
	VKTS string
}

func isfriend(id int, conf *Config) bool {
	for _, friend := range conf.TG.Friends {
		if friend == id {
			return true
		}
	}
	return false
}

var Conf Config
var Stat Status

func main() {
	file, e := ioutil.ReadFile("./conf.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	json.Unmarshal(file, &Conf)

	bot, err := tgbotapi.NewBotAPI(Conf.TG.API)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	ReadStatus()
	vkcli := vkapi.VKClient{APIKey: Conf.VK.APIkey, GroupID: Conf.VK.GroupID}
	vkcli.GetLongPollServer()
	if Stat.VKTS != "Null" {
		vkcli.TS = Stat.VKTS
	}
	go vkevent(bot, &Conf, &vkcli)
	go nestandart(bot, &Conf)
	go SaveStatus(&vkcli)

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if isfriend(update.Message.From.ID, &Conf) {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "sorry, papa told me to not talk with strangers")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
	}
}

func vkevent(bot *tgbotapi.BotAPI, conf *Config, vkcli *vkapi.VKClient) {
	for {
		updates, err := (vkcli.GetUpdates())
		if err != 0 {
			log.Printf("vk update err: %d", err)
		}
		if len(updates) != 0 {
			for _, v := range updates {
				var text string
				switch v.Type {
				case "group_join":
					text = fmt.Sprintf("id%d вступил в группу", v.EventObj.UID)
				case "group_leave":
					text = fmt.Sprintf("id%d вышел из группы", v.EventObj.UID)
				}
				msg := tgbotapi.NewMessage(conf.TG.Channel, text)
				bot.Send(msg)
			}
		}
	}
}

func SaveStatus(vkcli *vkapi.VKClient) {
	for {
		if vkcli.TS != Stat.VKTS {
			Stat.VKTS = vkcli.TS
			st, _ := json.MarshalIndent(Stat, "", " ")
			err := ioutil.WriteFile(Conf.StatusFile, st, 0644)
			if err != nil {
				log.Print(err)
			}
		} else {
			log.Print("SaveStatus: no new events")
		}
		duration := time.Duration(Conf.Save) * time.Minute
		time.Sleep(duration)
	}
}

func ReadStatus() {
	file, e := ioutil.ReadFile(Conf.StatusFile)
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		Stat.VKTS = "Null"
	}

	json.Unmarshal(file, &Stat)
}
