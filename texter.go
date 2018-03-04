package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
	"regexp"
	"strconv"
)

type Config struct {
	TwilioAccountSid  string `json:"twilioAccountSid"`
	TwilioAuthToken   string `json:"twilioAuthToken"`
	IntervalInSeconds int    `json:"intervalInSeconds"`
	NumberTo          string `json:"numberTo"`
	NumberFrom        string `json:"numberFrom"`
	TopThreshold	  int	 `json:"topThreshold"`
}

func RunGPUTempCommand() int {
	out, err := exec.Command("nvidia-smi", "--query-gpu=temperature.gpu", "--format=csv,noheader").Output()
	log.Println(string(out[:]))
	if err != nil {
		log.Fatal(err)
	}
	toString := string(out[:])
	reg, _ := regexp.Compile("[^a-zA-Z0-9]")
	toString = reg.ReplaceAllString(toString, "")
	i, _ := strconv.Atoi(toString)
	return i
}

// This function gets the configuration from the file
func GetConfigFromFile() Config {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(file)
	config := Config{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func SendGPUTempText(config Config) bool {
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + config.TwilioAccountSid + "/Messages"
	gpuTemp := RunGPUTempCommand()
	log.Println(gpuTemp < config.TopThreshold)
	if gpuTemp < config.TopThreshold {
		return false
	}
	messageBody := "Your GPU's current temperature is above your set threshold " + strconv.Itoa(config.TopThreshold) + " Celsius and is at " + strconv.Itoa(gpuTemp) + " degrees Celsius"
	msgData := url.Values{}
	msgData.Set("To", config.NumberTo)
	msgData.Set("From", config.NumberFrom)
	msgData.Set("Body", messageBody)
	msgDataReader := *strings.NewReader(msgData.Encode())
	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &msgDataReader)
	req.SetBasicAuth(config.TwilioAccountSid, config.TwilioAuthToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := client.Do(req)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var data map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&data)
		if err == nil {
			log.Println(data["sid"])
			return false
		}
	} else {
		log.Println(resp.Status)
		return false
	}
	return true
}
func main() {
	config := GetConfigFromFile()
	for {
		success := SendGPUTempText(config)
		if success {
			log.Println("Sent successfully")
		} else {
			log.Println("Failed to send")
		}
		log.Println("Waiting for specified time...")
		time.Sleep(time.Duration(config.IntervalInSeconds) * time.Second)
	}
}
