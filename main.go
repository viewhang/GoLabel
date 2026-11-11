package main

import (
	"log"

	"github.com/pengh/GoLabel/internal/editor"
)

// main 启动 GoLabel 并记录启动错误
func main() {
	if err := editor.Run(); err != nil {
		log.Fatal(err)
	}
}
