package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

//Idea is to display BTC price in the CLI. Maybe expand to display any instruments price.

var authtestsuccesscode int = 0 //0 means nogo, 1 means successful
var apitesturl string           //Set in Init function, will hold our secret key and be used to check
var stockpickclean string = ""  //not used globally, just used to pass right now
var cfg config                  //global variable to hold our config API

type config struct {
	APIkey string `json:"apikey"` //Needs to be uppercase as lower means unexported field - I refer to the type name in this struct, not the json
}
type TickerDetailsResponse struct {
	Results tickerdetails `json:"results"`
	Status  string        `json:"status"` //OK or NOT_FOUND possible
}
type tickerdetails struct {
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
}

func init() {
	//Special function to run before main() to load config (Special to GO! Language)
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
	responsebodystring := string(body) //Create new variable for the body

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
		getstock(stockpickclean) //Pass value of cleaned stockpick elsewhere
	} else {
		fmt.Println("Re-enter the ticker")
		userinput()
	}
}

func getstock(stockpickclean string) {
	fmt.Printf("Coming home. getting stock: %s using API Key: %s...\n", stockpickclean, cfg.APIkey+"...")
	//How would we check to see if a stock is a real stock or instrument? The eternal battle rages on

	ticketoverviewURL := fmt.Sprintf("https://api.polygon.io/v3/reference/tickers/%s?apiKey=%s", stockpickclean, cfg.APIkey)

	resp, err := http.Get(ticketoverviewURL)
	if err != nil {
		fmt.Printf("Error getting stock info for %s: %v\n", stockpickclean, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body for %s: %v\n", stockpickclean, err)
		return
	}

	var tickerresponse TickerDetailsResponse
	err = json.Unmarshal(body, &tickerresponse)
	if err != nil {
		fmt.Printf("Error parsing JSON response for %s: %v\n", stockpickclean, err)
		fmt.Printf("Raw response: %s\n", string(body)) //printing raw body just in case
		return
	}
	if tickerresponse.Status == "OK" && tickerresponse.Results.Ticker != "" {
		fmt.Printf("\n--- Stock Information for %s ---\n", tickerresponse.Results.Ticker)
		fmt.Printf("Company name: %s\n", tickerresponse.Results.Name)
		//fmt.Printf(ticketoverviewURL) //test
		generatestockchart(stockpickclean) //Call chart Gen Function

	} else if tickerresponse.Status == "NOT_FOUND" {
		fmt.Printf("Error: Financial Instrument '%s' not found or data found.\n", stockpickclean)
	} else {
		fmt.Printf("Could not retrieve '%s'. Status: %s\n", stockpickclean, &tickerresponse.Status)
	}

}

type AggregatesResponse struct {
	Ticker       string      `json:"ticker"`
	QueryCount   int         `json:"queryCount"`
	ResultsCount int         `json:"resultsCount"`
	Adjusted     bool        `json:"adjusted"`
	Results      []Aggregate `json:"results"`
	Status       string      `json:"status"`
	RequestID    string      `json:"request_id"`
	Count        int         `json:"count"`
}

type Aggregate struct {
	Open         float64 `json:"o"`
	Close        float64 `json:"c"`
	High         float64 `json:"h"`
	Low          float64 `json:"l"`
	Volume       float64 `json:"v"`
	Timestamp    int64   `json:"t"` // Unix Millisecond Timestamp
	VWAP         float64 `json:"vw"`
	Transactions int     `json:"n"`
}

func generatestockchart(stockpickclean string) {
	fmt.Printf("Fetching daily aggro data for %s and generating chart...\n", stockpickclean)
	//define range: 3 months from Today
	endDate := time.Now()
	startDate := endDate.AddDate(0, -3, 0)

	//fmt
	from := startDate.Format("2006-01-02")
	to := endDate.Format("2006-01-02")

	//Create API URL
	polygonAPIURL := fmt.Sprintf("https://api.polygon.io/v2/aggs/ticker/%s/range/1/day/%s/%s?adjusted=true&sort=asc&limit=5000&apiKey=%s", stockpickclean, from, to, cfg.APIkey)

	resp, err := http.Get(polygonAPIURL)
	if err != nil {
		fmt.Printf("Error getting aggro for %s: %v\n", stockpickclean, err)
		return
	}
	defer resp.Body.Close()
	//todo create chart or tie in complex GUI ideally. Lets try just normal
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body for %s: %v\n", stockpickclean, err)
		return
	}

	var aggsresponse AggregatesResponse
	err = json.Unmarshal(body, &aggsresponse)
	if err != nil {
		fmt.Printf("Error parsing JSON response for %s: %v\n", stockpickclean, err)
		fmt.Printf("Raw response: %s\n", string(body))
		return
	}

	if aggsresponse.Status != "OK" || len(aggsresponse.Results) == 0 {
		fmt.Printf("No Aggregate data found for '%s' or API not OK. Status: %s\n", stockpickclean, aggsresponse.Status)
		if aggsresponse.Status == "ERROR" {
			fmt.Printf("API error - %s\n", string(body))
		}
		return
	}
	//prepare data for go-echarts generation attempt
	var xData []string
	var klineData []opts.KlineData
	var volumeData []opts.BarData

	for _, agg := range aggsresponse.Results {
		//convert unix millis timstamp to date string
		t := time.Unix(0, agg.Timestamp*int64(time.Millisecond))
		xData = append(xData, t.Format("2006-01-02")) //fmt

		//order for cline to work
		klineData = append(klineData, opts.KlineData{Value: []float64{agg.Open, agg.Close, agg.Low, agg.High}})
		//then volume chart
		volumeData = append(volumeData, opts.BarData{Value: agg.Volume})
	}

	//Finally create our cline chart
	kline := charts.NewKLine()
	kline.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    fmt.Sprintf("%s Daily Candlestick Chart", stockpickclean),
			Subtitle: fmt.Sprintf("Data from %s to %s", from, to),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type:      "category",
			Data:      xData,
			AxisLabel: &opts.AxisLabel{Show: opts.Bool(true), Rotate: 45}, // Rotate
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type:  "value",
			Scale: opts.Bool(true), //AutoScale Y-axis
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "inside",
			Start:      50, //Start zoom and 50%
			End:        100,
			XAxisIndex: []int{0, 1}, //Apply to both kline and Vol xray
		}),
		charts.WithGridOpts(opts.Grid{Bottom: "10%"}), //Give space for zoomy slider

		charts.WithInitializationOpts(opts.Initialization{
			PageTitle: fmt.Sprintf("%s Chart", stockpickclean),
			//Theme:     opts.ThemeRiverData, //could try ThemeWesteros too
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger: "axis",
			AxisPointer: &opts.AxisPointer{
				Type: "cross",
			},
		}),
	)
	//Add Kline Series
	kline.AddSeries(fmt.Sprintf("%s OHLC", stockpickclean), klineData).
		SetSeriesOptions(
			charts.WithItemStyleOpts(opts.ItemStyle{
				BorderWidth: 1.5, // Make candles slightly thinner
			}),
		)

	// Create a separate Bar chart for Volume
	volumeBar := charts.NewBar()
	volumeBar.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Type:      "category",
			Data:      xData,
			GridIndex: 1,                // Connect to second grid
			Show:      opts.Bool(false), // Hide X-axis labels for volume chart (shares with Kline)
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type:        "value",
			GridIndex:   1, // Connect to second grid
			SplitNumber: 3, // Fewer ticks for volume Y-axis
		}),
		charts.WithGridOpts(opts.Grid{
			Top:    "80%", // Position volume chart below Kline
			Left:   "10%",
			Right:  "10%",
			Height: "10%", // Small height for volume
		}),
	)
	volumeBar.AddSeries("Volume", volumeData).
		SetSeriesOptions(
			charts.WithBarChartOpts(opts.BarChart{
				BarWidth: "60%", // Make volume bars wider
			}),
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: "#7f7f7f", // Gray color for volume bars
			}),
		)

	// Combine charts into a Page (useful for multiple charts on one page)
	//	page := charts.NewPage()
	//	page.AddCharts(kline, volumeBar) // Add both charts to the page

	// Render the chart to an HTML file
	//	outputFile := fmt.Sprintf("%s_kline_chart.html", strings.ToUpper(stockpickclean))
	//	f, err := os.Create(outputFile)
	//	if err != nil {
	//		fmt.Printf("Error creating HTML file %s: %v\n", outputFile, err)
	//		return
	//	}
	//	defer f.Close()

	//	err = page.Render(f)
	//	if err != nil {
	//		fmt.Printf("Error rendering chart to HTML: %v\n", err)
	//		return
	//	}

	//	fmt.Printf("Successfully generated chart for %s: %s\n", stockpickclean, outputFile)
	//	fmt.Printf("Open '%s' in your web browser to view the chart.\n", outputFile)

}
