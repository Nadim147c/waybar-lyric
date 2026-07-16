package config

import "time"

var (
	PrintInit       = false
	NoTooltip       = false
	PrintVersion    = false
	ToggleState     = false
	Verbose         = false
	Quiet           = false
	LyricOnly       = false
	Compact         = false
	Detailed        = false
	BreakTooltip    = 100000
	MaxTextLength   = 150
	TooltipLines    = 8
	TooltipColor    = "#cccccc"
	PlayerList      = []string{}
	FilterProfanity = false
	LogFilePath     = ""
	UpdateInterval  = time.Second / 4

	FilterProfanityType = ""

	Version string
)
