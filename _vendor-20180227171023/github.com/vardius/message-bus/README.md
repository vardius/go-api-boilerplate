Vardius - message-bus
================
[![Build Status](https://travis-ci.org/vardius/message-bus.svg?branch=master)](https://travis-ci.org/vardius/message-bus)
[![Go Report Card](https://goreportcard.com/badge/github.com/vardius/message-bus)](https://goreportcard.com/report/github.com/vardius/message-bus)
[![codecov](https://codecov.io/gh/vardius/message-bus/branch/master/graph/badge.svg)](https://codecov.io/gh/vardius/message-bus)
[![](https://godoc.org/github.com/vardius/message-bus?status.svg)](http://godoc.org/github.com/vardius/message-bus)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/vardius/message-bus/blob/master/LICENSE.md)

Go simple async message bus.

ABOUT
==================================================
Contributors:

* [Rafał Lorenz](http://rafallorenz.com)

Want to contribute ? Feel free to send pull requests!

Have problems, bugs, feature ideas?
We are using the github [issue tracker](https://github.com/vardius/message-bus/issues) to manage them.

HOW TO USE
==================================================

1. [GoDoc](http://godoc.org/github.com/vardius/message-bus)

## Basic example
```go
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
```

License
-------

This package is released under the MIT license. See the complete license in the package:

[LICENSE](LICENSE.md)
