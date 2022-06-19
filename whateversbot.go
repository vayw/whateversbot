package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	vkapi "github.com/vayw/gosocial"
)

type Config struct {
	VK           VKconf
	TG           TGconf
	StatusFile   string
	Save         int
	PollInterval int
}

type VKconf struct {
	APIkey  string
	GroupID string
}

type TGconf struct {
	API     string
	Friends []int64
	Channel int64
}

type Status struct {
	VKTS string
}

func isfriend(id int64, conf *Config) bool {
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
	go SaveStatus(&vkcli)

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		botAnswer(update.Message, bot, &Conf, &vkcli)
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
	}
}

func botAnswer(msg *tgbotapi.Message, bot *tgbotapi.BotAPI, conf *Config,
	vkcli *vkapi.VKClient) {
	newmsg := tgbotapi.NewMessage(msg.Chat.ID, "")
	if msg.IsCommand() {
		switch msg.Command() {
		case "count":
			count, err := vkcli.MembersCount()
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

func vkevent(bot *tgbotapi.BotAPI, conf *Config, vkcli *vkapi.VKClient) {
	for {
		duration := time.Duration(Conf.PollInterval) * time.Minute
		updates, err := vkcli.GetUpdates()
		log.Print("::vkevent:: updates", updates)
		if err != nil {
			log.Printf("::vkevent:: update err: %d", err)
			_ = vkcli.GetLongPollServer()
			time.Sleep(duration)
			continue
		}
		if len(updates) != 0 {
			var text string
			for _, v := range updates {
				strid := strconv.Itoa(v.EventObj.UID)
				userinfo, err := vkcli.GetUserData(strid, "sex")
				if err != nil {
					log.Printf("::vkevent:: update err: %d", err)
					text = fmt.Sprintf("%d %s", v.EventObj.UID, v.Type)
				} else {
					text = get_action(v.Type, userinfo[0])
				}
				msg := tgbotapi.NewMessage(conf.TG.Channel, text)
				log.Printf("::vkevent:: message= %s", text)
				bot.Send(msg)
			}
		}
		time.Sleep(duration)
	}
}

func get_action(event_type string, userinfo vkapi.User) string {
	var text, action string
	switch event_type {
	case "group_join":
		if userinfo.Sex == 1 {
			action = "вступила в группу"
		} else {
			action = "вступил в группу"
		}
	case "group_leave":
		if userinfo.Sex == 1 {
			action = "вышла из группы"
		} else {
			action = "вышел из группы"
		}
	}
	text = fmt.Sprintf("%s %s %s\nhttps://vk.com/id%d", userinfo.FirstName,
		userinfo.LastName, action, userinfo.UID)

	return text
}

func SaveStatus(vkcli *vkapi.VKClient) {
	for {
		if (vkcli.TS != Stat.VKTS) && (vkcli.TS != "") {
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
