package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	vkapi "github.com/vayw/gosocial"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
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
		default:
			newmsg.Text = "комманды: /count"
		}
	} else {
		if isfriend(msg.From.ID, conf) {
			newmsg.Text = msg.Text
			newmsg.ReplyToMessageID = msg.MessageID
		} else {
			newmsg.Text = "извините, но мы не друзья"
			newmsg.ReplyToMessageID = msg.MessageID
		}
	}
	bot.Send(newmsg)
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
				userinfo, err := vkcli.GetUserData(strid, "photo_100,sex")
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
	text = fmt.Sprintf("(id%d) %s %s %s\n%s", userinfo.UID, userinfo.FirstName,
		userinfo.LastName, action, userinfo.Photo100)

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
