/*
Package messagebus provides simple async message publisher

Basic example:
	package main

	import (
		"fmt"

		"github.com/vardius/message-bus"
	)

	func main() {
		bus := messagebus.New()

		var wg sync.WaitGroup
		wg.Add(2)

		bus.Subscribe("topic", func(v bool) {
			defer wg.Done()
			fmt.Println(v)
		})

		bus.Subscribe("topic", func(v bool) {
			defer wg.Done()
			fmt.Println(v)
		})

		bus.Publish("topic", true)
		wg.Wait()
	}
*/
package messagebus
