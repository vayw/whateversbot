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
var VKcli vkapi.VKClient

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

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	if Conf.VK.APIkey != "" {
		ReadStatus()
		VKcli := vkapi.VKClient{APIKey: Conf.VK.APIkey, GroupID: Conf.VK.GroupID}
		VKcli.GetLongPollServer()
		if Stat.VKTS != "Null" {
			VKcli.TS = Stat.VKTS
		}
		go vkevent(bot, &Conf)
		go SaveStatus()
	}

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		switch {
		case update.CallbackQuery != nil:
			BotCallback(update.CallbackQuery, bot)
		default:
			BotAnswer(update.Message, bot, &Conf)
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		}
	}
}

func vkevent(bot *tgbotapi.BotAPI, conf *Config) {
	for {
		duration := time.Duration(Conf.PollInterval) * time.Minute
		updates, err := VKcli.GetUpdates()
		log.Print("::vkevent:: updates", updates)
		if err != nil {
			log.Printf("::vkevent:: update err: %d", err)
			_ = VKcli.GetLongPollServer()
			time.Sleep(duration)
			continue
		}
		if len(updates) != 0 {
			var text string
			for _, v := range updates {
				strid := strconv.Itoa(v.EventObj.UID)
				userinfo, err := VKcli.GetUserData(strid, "sex")
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

func SaveStatus() {
	for {
		if (VKcli.TS != Stat.VKTS) && (VKcli.TS != "") {
			Stat.VKTS = VKcli.TS
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
