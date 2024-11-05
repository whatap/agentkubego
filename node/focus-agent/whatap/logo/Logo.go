package logo

import (
	"fmt"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/ansi"
)

func Print(version string) {
	fmt.Println(ansi.Green(""))
	fmt.Println(ansi.Green("    ______                WHATAP"))
	fmt.Println(ansi.Green("   / ____/___  _______  _______"))
	fmt.Println(ansi.Green("  / /_  / __ \\/ ___/ / / / ___/"))
	fmt.Println(ansi.Green(" / __/ / /_/ / /__/ /_/ (__  ) "))
	fmt.Println(ansi.Green("/_/    \\____/\\___/\\__,_/____/"))
	fmt.Println(ansi.Green("                                "))
	fmt.Printf(ansi.Green(" WhaTap Focus ver %s                   \n"), version)
	fmt.Println(ansi.Green(" Copyright ⓒ 2019 WhaTap Labs Inc. All rights reserved.\n"))
}
