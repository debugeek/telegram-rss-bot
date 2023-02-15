package main

import (
	"log"
	"math/rand"
	"time"
)

type Monitor struct {
	observers map[string]map[int64]*Observer
	ticker    *time.Ticker
	quit      chan bool
}

type Observer struct {
	identifier int64
	handler    func(items []*Item)
}

func InitMonitor() {
	monitorOnce.Do(func() {
		monitor = &Monitor{
			observers: make(map[string]map[int64]*Observer),
		}
	})
	monitor.Run()
	log.Println(`Monitor initialized`)
}

func (monitor *Monitor) Run() {
	go monitor.RunLoop()
}

func (monitor *Monitor) Stop() {
	monitor.quit <- true
}

func (monitor *Monitor) RunLoop() {
	monitor.refresh()

	monitor.ticker = time.NewTicker(time.Duration(rand.Intn(60)) * time.Second)
	monitor.quit = make(chan bool)

	for {
		select {
		case <-monitor.quit:
			return
		case <-monitor.ticker.C:
			monitor.refresh()
		}
	}
}

func (monitor *Monitor) AddObserver(observer *Observer, link string) {
	observers := monitor.observers[link]
	if observers == nil {
		observers = make(map[int64]*Observer)
		monitor.observers[link] = observers
	}
	observers[observer.identifier] = observer

	monitor.refresh()
}

func (monitor *Monitor) RemoveObserver(identifier int64, link string) {
	observers := monitor.observers[link]
	if observers == nil {
		return
	}

	delete(observers, identifier)
	monitor.observers[link] = observers
}

func (monitor *Monitor) refresh() {
	for link, observers := range monitor.observers {
		if len(observers) == 0 {
			continue
		}

		_, items, err := FetchItems(link)
		if len(items) == 0 || err != nil {
			continue
		}

		for _, observer := range observers {
			if observer.handler == nil {
				continue
			}
			observer.handler(items)
		}
	}

}
