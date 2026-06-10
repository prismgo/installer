package main

import (
	"prismgo/bootstrap"

	"github.com/prismgo/framework/console"
)

func main() {
	app := bootstrap.New()
	console.Run(app)
}
