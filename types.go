package main

type Profile struct {
	Cape string `json:"cape"`
	Capes any `json:"capes"`
	Id string `json:"id"`
	Name string `json:"name"`
	Skin any `json:"skin"`
}

type Ygg struct {
	Extra any `json:"extra"`
	Exp int `json:"exp"`
	Iat int `json:"iat"`
	Token string `json:"token"`
}

type Account struct {
	Active *bool `json:"active"`
	Entitlement any `json:"entitlement"`
	Msa any `json:"msa"`
	MsaClientId string `json:"msa-client-id"`
	Profile Profile `json:"profile"`
	Type string `json:"type"`
	Utoken any `json:"utoken"`
	XrpMain any `json:"xrp-main"`
	XrpMc any `json:"xrp-mc"`
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

type VerifiedAsset struct {
	Result bool
	Path string
	Asset Asset
}

type Assets struct {
	Objects map[string]Asset `json:"objects"`
}

type ServerId struct {
	Id string `json:"serverId"`
	Duration int `json:"expiresIn"`
}

type NoriskMod struct {
	Id string `json:"id"`
	Name string `json:"displayName"`
	Source map[string]string `json:"source"`
	Compatibility map[string]map[string]map[string]string `json:"compatibility"`
}

type Loader struct {
	Default map[string]map[string]string `json:"default"`
	Minecraft []string `json:"byMinecraft"`
}

type Pack struct {
	Name string `json:"displayName"`
	Desc string `json:"description"`
	Inherits []*string `json:"inheritsFrom"`
	Exclude []*string `json:"excludeMods"`
	Mods []NoriskMod `json:"mods"`
	Assets []string `json:"assets"`
	Experimental bool `json:"isExperimental"`
	Auto_update bool `json:"autoUpdate"`
	Loader *Loader `json:"loaderPolicy"`
}

type Versions struct {
	Packs map[string]Pack `json:"packs"`
	Repositories map[string]string `json:"repositories"`
}

type ModFile struct {
	Hashes map[string]string `json:"hashes"`
	Url string `json:"url"`
	Filename string `json:"filename"`
	Primary bool `json:"primary"`
	Size int `json:"size"`
	File_type *string `json:"file_type"`
}

type ModrinthMod struct {
	Versions []string `json:"game_versions"`
	Loaders []string `json:"loaders"`
	Id string `json:"id"`
	Project_id string `json:"project_id"`
	Author_id string `json:"author_id"`
	Featured bool `json:"featured"`
	Name string `json:"name"`
	Version string `json:"version_number"`
	Changelog string `json:"changelog"`
	Changelog_url *string `json:"changelog_url"`
	Date string `json:"date_published"`
	Downloads int `json:"downloads"`
	Version_type string `json:"version_type"`
	Status string `json:"status"`
	Requested_status *string `json:"requested_status"`
	Files []ModFile `json:"files"`
	Dependencies any `json:"dependencies"`
}

type ModEntry struct {
	Hash string
	Version string
	Id string
	filename string
	OldFile string
	SourceType string
	RepositoryRef string
	GroupId string
	ModrinthId string
	MavenId string
}
