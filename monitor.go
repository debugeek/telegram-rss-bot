package main

import (
	"time"
)

func (monitor *Monitor) runLoop() {
	monitor.refresh()

	monitor.ticker = time.NewTicker(time.Duration(60) * time.Second)
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

func (monitor *Monitor) addObserver(observer *Observer, link string) {
	observers := monitor.observers[link]
	if observers == nil {
		observers = make(map[int64]*Observer)
		monitor.observers[link] = observers
	}
	observers[observer.identifier] = observer

	monitor.refresh()
}

func (monitor *Monitor) removeObserver(identifier int64, link string) {
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

		_, items, err := fetchItems(link)
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
