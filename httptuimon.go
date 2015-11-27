package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gizak/termui"
)

type Monitor struct {
	URL      string
	Healthy  bool `json:"-"`
	Duration time.Duration
}

type input struct {
	Monitors []Monitor
}

func (m *Monitor) check() error {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", m.URL, nil)
	if err != nil {
		log.Printf("Unable to create request for %v\n", m.URL)
		return err
	}
	start := time.Now()
	response, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to perform request to %v\n", m.URL)
		return err
	}
	m.Duration = time.Since(start)
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		m.Healthy = true
	} else {
		m.Healthy = false
	}

	return nil
}

func main() {
	config := flag.String("config", "config.json", "Configuration file")
	flag.Parse()
	file, err := ioutil.ReadFile(*config)
	if err != nil {
		log.Fatalf("Unable to read config file: %v\n", err)
	}
	var in input
	err = json.Unmarshal(file, &in)
	if err != nil {
		log.Fatalf("Unable to read config: %v\n", err)
	}
	err = termui.Init()
	if err != nil {
		log.Fatalf("Unable to init termui: %v\n", err)
	}
	defer termui.Close()
	list := termui.NewList()
	list.Width = 20
	var urls []string
	for i, m := range in.Monitors {
		e := fmt.Sprintf("[%v] %v", i, m.URL)
		urls = append(urls, e)
		if list.Width < int(float64(len(e))*float64(1.5)) {
			list.Width = int(float64(len(e)) * float64(1.5))
		}
	}
	list.Items = urls
	list.ItemFgColor = termui.ColorYellow
	list.BorderLabel = "URLs"
	list.Height = len(urls) * 2
	list.Y = 0
	list.X = 0

	sp := termui.NewSparklines()
	sp.BorderLabel = "Response times"
	sp.Y = list.Height
	sp.X = 0
	sp.Height = (len(urls)*3 - 1)
	sp.Width = list.Width
	for i, _ := range urls {
		spark := termui.Sparkline{}
		spark.Height = 1
		spark.Title = fmt.Sprintf("URL %v", i)
		spark.LineColor = termui.ColorCyan
		spark.TitleColor = termui.ColorWhite
		sp.Add(spark)
	}
	for i, m := range in.Monitors {
		m.check()
		sp.Lines[i].Data = append(sp.Lines[i].Data, int(m.Duration))
	}

	termui.Render(list, sp)

	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})

	termui.Handle("/timer/1s", func(e termui.Event) {
		t := e.Data.(termui.EvtTimer)
		if t.Count%5 == 0 {
			for i, m := range in.Monitors {
				if err := m.check(); err != nil {
					log.Fatalf("%v\n", err)
				}
				sp.Lines[i].Data = append(sp.Lines[i].Data, int(m.Duration))
			}
			termui.Render(list, sp)
		}
	})

	termui.Loop()

}
