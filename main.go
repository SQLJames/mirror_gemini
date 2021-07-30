package main

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "Crypto"

func main() {
	//http.Handle("/metrics", promhttp.Handler())
	//log.Fatal(http.ListenAndServe(":9101", nil))

	e := NewExporter()
	prometheus.MustRegister(e)

	http.Handle(metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Crypto Channel Exporter</title></head>
             <body>
             <h1>Crypto Channel Exporter</h1>
             <p><a href='` + metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}

//Exporter holds all details associated to the exporter
type Exporter struct {
	BaseURL string
}

/*
GET https://api.gemini.com/v2/ticker/:symbol
{
  "symbol": "BTCUSD",
  "open": "9121.76",
  "high": "9440.66",
  "low": "9106.51",
  "close": "9347.66",
  "changes": [
    "9365.1",
    "9386.16",
    "9373.41",
    "9322.56",
    "9268.89",
    "9265.38",
    "9245",
    "9231.43",
    "9235.88",
    "9265.8",
    "9295.18",
    "9295.47",
    "9310.82",
    "9335.38",
    "9344.03",
    "9261.09",
    "9265.18",
    "9282.65",
    "9260.01",
    "9225",
    "9159.5",
    "9150.81",
    "9118.6",
    "9148.01"
  ],
  "bid": "9345.70",
  "ask": "9347.67"
}
*/

type V2TickerResponse struct {
	Symbol  string   `json:"symbol"`
	Open    string   `json:"open"`
	High    string   `json:"high"`
	Low     string   `json:"low"`
	Close   string   `json:"close"`
	Bid     string   `json:"bid"`
	Ask     string   `json:"ask"`
	Changes []string `json:"changes"`
}

/*GET https://api.gemini.com/v1/symbols

["btcusd","ethbtc","ethusd","zecusd","zecbtc","zeceth", "zecbch", "zecltc", "bchusd", "bchbtc", "bcheth", "ltcusd", "ltcbtc", "ltceth", "ltcbch", "batusd", "daiusd", "linkusd", "oxtusd", "batbtc", "linkbtc", "oxtbtc", "bateth", "linketh", "oxteth", "ampusd", "compusd", "paxgusd", "mkrusd", "zrxusd", "kncusd", "manausd", "storjusd", "snxusd", "crvusd", "balusd", "uniusd", "renusd", "umausd", "yfiusd", "btcdai", "ethdai", "aaveusd", "filusd", "btceur", "btcgbp", "etheur", "ethgbp", "btcsgd", "ethsgd"]


*/
type V1SymbolResponse struct {
	Symbol []string
}

var (
	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}

	listenAddress = ":9141"
	metricsPath   = "/metrics"

	// Metrics
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last query successful.",
		nil, nil,
	)
	open = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "opening_price"),
		"Open price from 24 hours ago (per Currency)",
		[]string{"currency"}, nil,
	)
	high = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "high_price"),
		"High price from 24 hours ago (per Currency).",
		[]string{"currency"}, nil,
	)
	low = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "low_price"),
		"Low price from 24 hours ago (per Currency).",
		[]string{"currency"}, nil,
	)
	close = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "close_price"),
		"Close price (most recent trade)(per Currency).",
		[]string{"currency"}, nil,
	)
	bid = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "bid_price"),
		"Current best bid (per Currency).",
		[]string{"currency"}, nil,
	)
	ask = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ask_price"),
		"Current best offer (per Currency).",
		[]string{"currency"}, nil,
	)
)

func NewExporter() *Exporter {
	return &Exporter{
		BaseURL: "https://api.gemini.com/",
	}
}

func (e *Exporter) LoadSymbols() ([]string, error) {
	req, err := http.NewRequest("GET", e.BaseURL+"v1/symbols", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	log.Println(string(body))

	var V1SymbolResponseJson V1SymbolResponse
	err = json.Unmarshal(body, &V1SymbolResponseJson.Symbol)
	if err != nil {
		return nil, err
	}

	return V1SymbolResponseJson.Symbol, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- open
	ch <- high
	ch <- low
	ch <- close
	ch <- bid
	ch <- ask
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	symbols, err := e.LoadSymbols()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0,
		)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1,
	)

	e.UpdateMetrics(symbols, ch)
}

func (e *Exporter) UpdateMetrics(Symbols []string, ch chan<- prometheus.Metric) {
	for _, Symbol := range Symbols {
		req, err := http.NewRequest("GET", e.BaseURL+"v2/ticker/"+Symbol, nil)
		if err != nil {
			log.Fatal(err)
		}

		// Make request and show output.
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		var TickerResponse V2TickerResponse
		// we unmarshal our byteArray which contains our
		// xmlFiles content into 'users' which we defined above
		err = json.Unmarshal(body, &TickerResponse)
		if err != nil {
			log.Fatal(err)
		}
		/*
			ch <- up
			ch <- open
			ch <- high
			ch <- low
			ch <- close
			ch <- bid
			ch <- ask
		*/
		Open, _ := strconv.ParseFloat(TickerResponse.Open, 64)
		High, _ := strconv.ParseFloat(TickerResponse.High, 64)
		Low, _ := strconv.ParseFloat(TickerResponse.Low, 64)
		Close, _ := strconv.ParseFloat(TickerResponse.Close, 64)
		Bid, _ := strconv.ParseFloat(TickerResponse.Bid, 64)
		Ask, _ := strconv.ParseFloat(TickerResponse.Ask, 64)
		ch <- prometheus.MustNewConstMetric(
			open, prometheus.GaugeValue, Open, Symbol,
		)
		ch <- prometheus.MustNewConstMetric(
			high, prometheus.GaugeValue, High, Symbol,
		)
		ch <- prometheus.MustNewConstMetric(
			low, prometheus.GaugeValue, Low, Symbol,
		)
		ch <- prometheus.MustNewConstMetric(
			close, prometheus.GaugeValue, Close, Symbol,
		)
		ch <- prometheus.MustNewConstMetric(
			bid, prometheus.GaugeValue, Bid, Symbol,
		)
		ch <- prometheus.MustNewConstMetric(
			ask, prometheus.GaugeValue, Ask, Symbol,
		)

	}

	log.Println("Endpoint scraped")
}
