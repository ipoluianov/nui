package main

import (
	"fmt"
	"os/exec"
	"time"
)

func main() {
	for i := 0; i < 5; i++ {
		exec.Command("true").Start() // what OpenURL does with xdg-open
	}
	time.Sleep(500 * time.Millisecond)
	out, _ := exec.Command("sh", "-c", "ps -o pid,stat,comm --ppid $PPID").Output()
	fmt.Print(string(out))
}
