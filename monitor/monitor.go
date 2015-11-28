package monitor

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"
)

type Monitor interface {
	Check(chan CheckUpdate, uint32)
	Healthy() bool
	Source() string
	Duration() time.Duration
}

type CheckUpdate struct {
	Id       uint32
	Healthy  bool
	Duration time.Duration
	Err      error
}

type HTTPMonitor struct {
	url      string
	healthy  bool
	duration time.Duration
}

func NewHTTPMonitor(url string) *HTTPMonitor {
	return &HTTPMonitor{url: url}
}

func (m *HTTPMonitor) Source() string {
	return m.url
}

func (m *HTTPMonitor) Healthy() bool {
	return m.healthy
}

func (m *HTTPMonitor) Duration() time.Duration {
	return m.duration
}

func (m *HTTPMonitor) Check(c chan CheckUpdate, id uint32) {
	start := time.Now()
	ret := CheckUpdate{Id: id}
	var err error = nil
	defer func() {
		ret.Duration = time.Since(start)
		ret.Err = err
		c <- ret
	}()
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", m.url, nil)
	if err != nil {
		log.Printf("Unable to create request for %v\n", m.url)
		return
	}
	response, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to perform request to %v\n", m.url)
		return
	}
	m.duration = time.Since(start)
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		m.healthy = true
	} else {
		m.healthy = false
	}
}
