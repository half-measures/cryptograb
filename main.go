package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

//Idea is to display BTC price in the CLI. Maybe expand to display any instruments price.

var authtestsuccesscode int = 0 //0 means nogo, 1 means successful
var apitesturl string           //Set in Init function, will hold our secret
var stockpickclean string = ""  //not used globally, just used to pass right now
var cfg config                  //global variable to hold our config API

type config struct {
	APIkey string `json:"apikey"` //Needs to be uppercase as lower means unexported field
}

func init() {
	//Special function to run before main() to load config (Special to GO!)
	//Obj is to load the json file in.
	configfile, err := os.Open("config.json")
	if err != nil {
		fmt.Printf("Error opening config.json: %v\n", err)
		fmt.Println("Ensure config.json exists in same dir as main.go")
		os.Exit(1)
	}
	defer configfile.Close()
	byteValue, _ := io.ReadAll(configfile) //Read our config
	err = json.Unmarshal(byteValue, &cfg)  //Unmarshal json into our config
	if err != nil {
		fmt.Printf("error parsing config.json: %v\n", err)
		os.Exit(1)
	}
	// lets create our URL and key for our check function
	apitesturl = "https://api.polygon.io/v3/reference/dividends?apiKey=" + cfg.APIkey
	//bufio.NewReader(os.Stdin).ReadBytes('\n') Seems to stop the program instead of pause but oh well
}

func main() {
	statuscode := authtest()

	if statuscode == 0 {
		fmt.Println("Auth has failed, check the API key and consult Polygon.com for more info")
		os.Exit(1)
	}
	// API key is good, continue
	fmt.Println("Auth passed, Proceeding...")
	userinput()

}

func authtest() int { //Test our connection to make sure we have a valid API key
	resp, err := http.Get(apitesturl)
	if err != nil {
		fmt.Printf("Error during API connection test: %v\n", err)
		return 0
	}
	defer resp.Body.Close() //Ensure closure

	//Read our response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error Reading Body: %v", err)
		return 0
	}
	responsebodystring := string(body)

	keyword := "results"
	if strings.Contains(responsebodystring, keyword) {
		fmt.Println("Success, Connection valid")
		authtestsuccesscode = 1
		return 1

	} else {
		fmt.Printf("Failure! Get out!\n")
		fmt.Printf("Response Body: %s", responsebodystring)
		authtestsuccesscode = 0
		return 0
	}
}

func userinput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter the Stock ticket you would like info on:")
	stockpick, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}
	//We now have stockpick
	stockpickclean := strings.TrimSpace(stockpick) //Clean whitespaces
	//We should check the stock to make sure its real and valid
	fmt.Printf("You have entered the stock '%s'. Is that correct? [Y or N]\n", stockpickclean)
	reader1 := bufio.NewReader(os.Stdin)
	inputcheck, err := reader1.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading confirm", err)
		return
	}
	yesnocheck := strings.TrimSpace(inputcheck)
	if strings.ToUpper(yesnocheck) == "Y" {
		isstockvalid(stockpickclean) //Pass value of cleaned stockpick elsewhere
	} else {
		fmt.Println("Re-enter the ticker")
		userinput()
	}
}

func isstockvalid(stockpickclean string) {
	fmt.Printf("Coming home. Validating stock: %s using API Key: %s...\n", stockpickclean, cfg.APIkey+"...")

}
