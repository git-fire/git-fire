package flavor

var startupFireQuotes = []string{
	"Fire walk with me.",
	"A light in the dark provides hope.",
	"We didn't start the fire, but we can back it up.",
	"This is fine. Everything is version controlled.",
	"Even the smallest spark can light the way home.",
	"May your commits be hot and your rollbacks rare.",
	"May the forge be with you.",
	"From ember to branch, every change deserves shelter.",
}

func RandomStartupFireQuote() string {
	return PickRandomString(startupFireQuotes)
}

func StartupFireQuotes() []string {
	return append([]string(nil), startupFireQuotes...)
}
