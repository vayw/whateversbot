package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

const URL = "https://lk.castnow.ru/public/10/current.json"

var logger = log.New(os.Stdout, "radiocast ", log.Ltime)

type APIResponse struct {
	Next   int64  `json:"next"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
}

func sleep(until int64) time.Duration {
	now := time.Now()
	next := time.Unix(until, 0)
	c := (next.Unix() - now.Unix()) + 3
	if c < 10 {
		c = 15
	}
	return (time.Duration(c) * time.Second)
}

func nestandart(bot *tgbotapi.BotAPI, conf *Config) {
	var info APIResponse
	for {
		resp, err := http.Get(URL)
		defer resp.Body.Close()
		if err != nil {
			logger.Print("[ERR} ", err)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		jsonErr := json.Unmarshal([]byte(body), &info)
		if jsonErr != nil {
			logger.Print("[ERR]", jsonErr)
		}
		if info.Artist == "Whatevers" {
			text := fmt.Sprintf("Radio Nestandart now playing %s - %s", info.Artist, info.Title)
			msg := tgbotapi.NewMessage(conf.TG.Channel, text)
			bot.Send(msg)
		}

		time.Sleep(sleep(info.Next))
	}
}
