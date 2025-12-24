package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: memhog <MB>")
		os.Exit(1)
	}

	var mb int
	fmt.Sscanf(os.Args[1], "%d", &mb)

	fmt.Printf("memhog: trying to allocate %d MB\n", mb)

	size := mb * 1024 * 1024
	buf := make([]byte, size)

	// chạm vào từng page để kernel thực sự cấp RAM
	for i := 0; i < len(buf); i += 4096 {
		buf[i] = 1
	}

	fmt.Printf("memhog: allocated %d MB successfully\n", mb)
	fmt.Println("sleeping...")

	time.Sleep(30 * time.Second)
}
