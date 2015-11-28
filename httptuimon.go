package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/gizak/termui"
	"github.com/jvgutierrez/httptuimon/monitor"
)

type configFile struct {
	Entries []configEntry `json:"monitors"`
}

type configEntry struct {
	URL string
}

type UIMonitor struct {
	index   uint32
	monitor monitor.Monitor
}

const (
	COLOR_OK = termui.ColorGreen
	COLOR_KO = termui.ColorRed
)

func readConfig(fn string) []UIMonitor {
	file, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatalf("Unable to read config file: %v\n", err)
	}
	var cf configFile
	err = json.Unmarshal(file, &cf)
	if err != nil {
		log.Fatalf("Unable to read config: %v\n", err)
	}
	ret := make([]UIMonitor, len(cf.Entries))
	for i, entry := range cf.Entries {
		ret[i].index = uint32(i)
		ret[i].monitor = monitor.NewHTTPMonitor(entry.URL)
	}
	return ret
}

func main() {
	configFile := flag.String("config", "config.json", "Configuration file")
	flag.Parse()
	monitors := readConfig(*configFile)
	updates := make(chan monitor.CheckUpdate)
	err := termui.Init()
	if err != nil {
		log.Fatalf("Unable to init termui: %v\n", err)
	}
	defer termui.Close()
	list := termui.NewList()
	list.Width = 20
	var urls []string
	for _, m := range monitors {
		e := fmt.Sprintf("[%v] %v", m.index, m.monitor.Source())
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
	for _, m := range monitors {
		go m.monitor.Check(updates, m.index)
	}
loop:
	for {
		select {
		case u := <-updates:
			if u.Healthy && u.Err == nil {
				sp.Lines[u.Id].LineColor = COLOR_OK
			} else {
				sp.Lines[u.Id].LineColor = COLOR_KO
			}
		case <-time.After(2 * time.Second):
			break loop
		}
	}

	termui.Render(list, sp)

	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})

	termui.Handle("/timer/1s", func(e termui.Event) {
		t := e.Data.(termui.EvtTimer)
		if t.Count%5 == 0 {
			for _, m := range monitors {
				go m.monitor.Check(updates, m.index)
			}
		loop:
			for {
				select {
				case u := <-updates:
					sp.Lines[u.Id].Data = append(sp.Lines[u.Id].Data, int(u.Duration))
					if u.Healthy && u.Err == nil {
						sp.Lines[u.Id].LineColor = COLOR_OK
					} else {
						sp.Lines[u.Id].LineColor = COLOR_KO
					}
				case <-time.After(2 * time.Second):
					break loop
				}
			}
			termui.Render(list, sp)
		}
	})

	termui.Loop()

}
