// package main

// import (
// 	"fmt"
// 	"os"
// 	"containergo/internal/container"
// )

// func main() {
// 	if len(os.Args) < 2 {
// 		fmt.Println("usage: container <run|child> <command...>")
// 		os.Exit(1)
// 	}

//		switch os.Args[1] {
//		case "run":
//			// tất cả args sau "run" là command cần chạy trong container
//			container.Run(os.Args[2:])
//		case "child":
//			// khi process được re-exec với "child" sẽ vào container.Child
//			container.Child(os.Args[2:])
//		default:
//			fmt.Println("unknown command:", os.Args[1])
//			os.Exit(1)
//		}
//	}
package main

import (
	"flag"
	"fmt"
	"os"

	"containergo/internal/cli"
	"containergo/internal/container"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: container <run|child|stats> [options]")
		os.Exit(1)
	}

	switch os.Args[1] {

	case "run":
		// tất cả args sau "run" là command cần chạy trong container
		container.Run(os.Args[2:])

	case "child":
		// process được re-exec để vào namespace
		container.Child(os.Args[2:])

	case "stats":
		statsCmd := flag.NewFlagSet("stats", flag.ExitOnError)
		watch := statsCmd.Bool("watch", false, "watch container stats in realtime")
		_ = statsCmd.Parse(os.Args[2:])

		cli.Stats(*watch)

	default:
		fmt.Println("unknown command:", os.Args[1])
		os.Exit(1)
	}
}
