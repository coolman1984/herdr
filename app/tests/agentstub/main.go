// A local transport fixture, not an AI CLI. It never runs tools or network calls.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("OpenAI Codex transport fixture (offline test only)\n\u203a") // Codex's empty input box (C7)
	reader := bufio.NewScanner(os.Stdin)
	reader.Buffer(make([]byte, 4096), 128<<10)
	for reader.Scan() {
		text := reader.Text()
		if strings.Contains(text, "BLOCK") {
			fmt.Println("Do you trust this folder?\nenter to confirm  esc to cancel")
			continue
		}
		fmt.Println("DECK-" + "AGENT-OK\n\u203a")
	}
	if err := reader.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
