// herdr-goal: give a goal, the engine finishes it. See docs/GOAL-ENGINE.md.
package main

import (
	"os"

	"herdrdeck/internal/goal"
)

func main() { os.Exit(goal.Main(os.Args[1:])) }
