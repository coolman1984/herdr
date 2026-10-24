package main

import (
	"embed"
	"herdrdeck/internal/deck"
)

//go:embed web/*
var assets embed.FS

func main() { deck.Main(assets) }
