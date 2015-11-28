package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/gizak/termui"
	"github.com/jvgutierrez/httptuimon/monitor"
)

type input struct {
	Monitors []monitor.Monitor
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
		e := fmt.Sprintf("[%v] %v", i, m.Source())
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
		m.Check()
		sp.Lines[i].Data = append(sp.Lines[i].Data, int(m.Duration()))
	}

	termui.Render(list, sp)

	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})

	termui.Handle("/timer/1s", func(e termui.Event) {
		t := e.Data.(termui.EvtTimer)
		if t.Count%5 == 0 {
			for i, m := range in.Monitors {
				if err := m.Check(); err != nil {
					log.Fatalf("%v\n", err)
				}
				sp.Lines[i].Data = append(sp.Lines[i].Data, int(m.Duration()))
			}
			termui.Render(list, sp)
		}
	})

	termui.Loop()

}
