package player

import (
	"fmt"
	"os/exec"
)

type Player struct {
	playerType string
	playerPath string
}

func New(playerType string, paths map[string]string) *Player {
	path, ok := paths[playerType]
	if !ok {
		path = playerType // fallback to using playerType as path
	}
	return &Player{
		playerType: playerType,
		playerPath: path,
	}
}

func (p *Player) Play(url string) error {
	var cmd *exec.Cmd

	switch p.playerType {
	case "mpv":
		cmd = exec.Command(p.playerPath, url)
	case "vlc":
		cmd = exec.Command(p.playerPath, url)
	default:
		return fmt.Errorf("unsupported player type: %s", p.playerType)
	}

	return cmd.Start()
}
