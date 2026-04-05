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
	"Light the beacons; your remotes are waiting.",
	"Not all who wander lack a remote.",
	"Help me, git-fire; you're my only hope for this workstation.",
	"Rebellions are built on branches. Push yours.",
	"When someone asks if you're backed up, you say yes.",
	"There is no spoon; there is a remote.",
	"Where we're going, we still need good commit messages.",
	"Today we cancel the data-loss apocalypse.",
	"Witness: your changes, delivered.",
	"Come with me if you want your work to live.",
	"Life finds a way onto the remote.",
	"Praise the sun, then praise the remote.",
	"It's dangerous to go solo; bring a backup branch.",
	"This was a triumph; your mirrors are a huge success.",
	"Another run? The underworld respects persistence.",
	"Respawn safe: set your remote and rest.",
	"The push must flow.",
	"Don't panic; there's almost certainly a branch for that.",
	"Journey before destination; mirror before disaster.",
	"Stay synced, squad; the graph reads hot.",
	"Welcome to the backup party, pal.",
	"It's alive, and it's on origin.",
	"From forge to fetch, keep the flame in two places.",
	"Free your mind; keep your commits off this machine alone.",
}

func RandomStartupFireQuote() string {
	return PickRandomString(startupFireQuotes)
}

func StartupFireQuotes() []string {
	return append([]string(nil), startupFireQuotes...)
}
