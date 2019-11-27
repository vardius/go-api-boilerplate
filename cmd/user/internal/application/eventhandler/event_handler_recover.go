package eventhandler

import (
	"log"
	"runtime/debug"
)

func recoverEventHandler() {
	if r := recover(); r != nil {
		log.Printf("[EventHandler] Recovered in %v\n%s\n", r, debug.Stack())
	}
}
