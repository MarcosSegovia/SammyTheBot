package weather

import (
	"io/ioutil"
	"net/http"
	"log"
	"bytes"
	"encoding/json"

	"github.com/danbondd/temperature/tempconv"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/MarcosSegovia/sammy-the-bot/sammy"
	"strconv"
	"fmt"
)

const (
	BARCELONA = "6356055"
)

type Weather struct {
	sammy *sammy.Sammy
	Cmd   *sammy.Cmd
}

type Response struct {
	Coord      map[string]float64 `json:"coord"`
	Conditions []map[string]interface{} `json:"weather"`
	Main       map[string]float64 `json:"main"`
	City       string `json:"name"`
}

func NewWeather(s *sammy.Sammy) *Weather {
	w := new(Weather)
	w.sammy = s
	w.Cmd = sammy.NewCommand("weather", "/weather", "Show current forecast")
	return w
}

var oldMsg *tgbotapi.Message

//return errors
func (w *Weather) Evaluate(msg *tgbotapi.Message) {
	if oldMsg == nil {
		if msg.Text != w.Cmd.Exec {
			return
		}
		oldMsg = msg
		respMsg := tgbotapi.NewMessage(msg.Chat.ID, "Would you like to get forecast from your current location or Barcelona?")
		lButton := tgbotapi.NewKeyboardButtonLocation("Current location")
		bButton := tgbotapi.NewKeyboardButton("Barcelona")
		buttons := tgbotapi.NewKeyboardButtonRow(lButton, bButton)
		keyboard := tgbotapi.NewReplyKeyboard(buttons)
		keyboard.OneTimeKeyboard = true
		respMsg.ReplyMarkup = keyboard
		_, err := w.sammy.Api.Send(respMsg)
		check(err, "could not send message because: %v")
		return
	}
	newMsg := tgbotapi.NewMessage(msg.Chat.ID, "Sorry, this does not fit here...")
	defer func() {
		keyboard := tgbotapi.NewRemoveKeyboard(false)
		newMsg.ReplyMarkup = keyboard
		_, err := w.sammy.Api.Send(newMsg)
		check(err, "could not send message because: %v")
	}()

	oldMsg = nil
	request := buildRequest(w, msg)
	if request == nil {
		return
	}
	resp, err := http.Get(request.String())
	defer resp.Body.Close()
	check(err, "could not get an appropriate response: %v")

	body, err := ioutil.ReadAll(resp.Body)
	wresp := Response{}
	err = json.Unmarshal(body, &wresp)
	check(err, "could not parse from json: %v")
	if len(wresp.Conditions) == 0 {
		fmt.Println("ERROR !")
		return
	}
	var buffer bytes.Buffer

	buffer.WriteString("Seems that we will have ")
	buffer.WriteString(wresp.Conditions[0]["main"].(string) + " ")
	switch  wresp.Conditions[0]["id"].(float64) {

	//Thunder
	case 200, 201, 202, 210, 211, 212, 221, 230, 231, 232:
		buffer.Write([]byte{226, 155, 136})

	//Drizzle
	case 300, 301, 302, 310, 311, 312, 313, 314, 321:
		buffer.Write([]byte{226, 155, 136})

	//Rain
	case 500, 501, 502, 503, 504, 511, 520, 521, 522, 531:
		buffer.Write([]byte{240, 159, 140, 167})

	//Snow
	case 600, 601, 602, 611, 612, 615, 616, 620, 621, 622:
		buffer.Write([]byte{226, 157, 132})

	//Fog
	case 701, 711, 721, 731, 741, 751, 761, 762, 771, 781:
		buffer.Write([]byte{240, 159, 140, 171})

	//Clear
	case 800:
		buffer.Write([]byte{226, 152, 128})

	//Few clouds
	case 801:
		buffer.Write([]byte{240, 159, 140, 164})

	//Clouds
	case 802, 803, 804:
		buffer.Write([]byte{226, 155, 133})

	}
	buffer.WriteString(" for today here in ")
	buffer.WriteString(wresp.City)
	buffer.WriteString(", with a temperature of ")
	celsius := tempconv.KelvinToCelcius(tempconv.Kelvin(wresp.Main["temp"]))
	buffer.WriteString(celsius.String())
	newMsg.Text = buffer.String()

}
func buildRequest(w *Weather, msg *tgbotapi.Message) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	buffer.WriteString("http://api.openweathermap.org/data/2.5/weather?appid=")
	buffer.WriteString(w.sammy.Brain.GetString("configuration.weather"))

	if msg.Location != nil {
		buffer.WriteString("&lat=" + strconv.FormatFloat(msg.Location.Latitude, 'f', 2, 64))
		buffer.WriteString("&lon=" + strconv.FormatFloat(msg.Location.Longitude, 'f', 2, 64))
		return buffer
	}
	if msg.Text == "Barcelona" {
		buffer.WriteString("&id=" + BARCELONA)
		return buffer
	}
	return nil
}

func (w *Weather) Description() string {
	return w.Cmd.Exec + " - " + w.Cmd.Desc
}

func check(err error, msg string) {
	if err != nil {
		log.Printf(msg, err)
	}
}
