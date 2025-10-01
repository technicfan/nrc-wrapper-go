package main

type Profile struct {
	Id string `json:"id"`
	Name string `json:"name"`
}

type Ygg struct {
	Token string `json:"token"`
}

type Account struct {
	Active any `json:"active"`
	Profile Profile `json:"profile"`
	Type string `json:"type"`
	Ygg Ygg `json:"ygg"`
}

type PrismData struct {
	Accounts []Account `json:"accounts"`
	FormatVersion int `json:"formatVersion"`
}

type Asset struct {
	Hash string `json:"hash"`
	Size int `json:"size"`
}

type Assets struct {
	Objects map[string]Asset `json:"objects"`
}

type ServerId struct {
	Id string `json:"serverId"`
}

type NoriskMod struct {
	Id string `json:"id"`
	Name string `json:"displayName"`
	Source map[string]string `json:"source"`
	Compatibility map[string]map[string]map[string]any `json:"compatibility"`
}

type Pack struct {
	Name string `json:"displayName"`
	Desc string `json:"description"`
	Inherits []string `json:"inheritsFrom"`
	Exclude []any `json:"excludeMods"`
	Mods []NoriskMod `json:"mods"`
	Assets []string `json:"assets"`
}

type Versions struct {
	Packs map[string]Pack `json:"packs"`
	Repositories map[string]string `json:"repositories"`
}

type Components struct {
	CName string `json:"cachedName"`
	Version string `json:"version"`
}

type PrismInstance struct {
	Components []Components `json:"components"`
}

type ModEntry struct {
	Hash string
	Version string
	Id string
	Filename string
	OldFile string
	SourceType string
	RepositoryRef string
	GroupId string
	ModrinthId string
	ProjectSlug string
	MavenId string
}
