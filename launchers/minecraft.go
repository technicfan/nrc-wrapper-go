package launchers

import (
	"errors"
)

type Minecraft struct {
	Profile       string
	Version       string
	Loader        string
	LoaderVersion string
	Username      string
	Uuid          string
	Token         string
}

func Get_minecraft_details(
	path string,
	launcher string,
) (Minecraft, error) {
	switch launcher {
	case "prism":
		return get_prism_details(path)
	case "modrinth":
		return get_modrinth_details(path)
	default:
		return Minecraft{}, errors.New("Minecraft details not found")
	}
}
