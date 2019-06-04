package application

import "log"

func recoverEventHandler() {
	if r := recover(); r != nil {
		log.Printf("[EventHandler] Recovered in %v", r)
	}
}
