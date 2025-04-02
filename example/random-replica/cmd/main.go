package main

import (
	"fmt"
	"log"
	"math/rand/v2"
)

func main() {
	log.Println("Start...")
	replica := rand.IntN(10)
	fmt.Print(replica)
}
