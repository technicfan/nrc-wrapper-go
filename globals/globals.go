package globals

var NORISK_API_URL = "https://api.norisk.gg/api/v1"
var MAIN_PACKS = []string{"norisk-prod", "norisk-bughunter", "norisk-development"}
var REFRESH bool

const (
	NORISK_API_STAGING_URL = "https://api-staging.norisk.gg/api/v1"
	NORISK_ASSETS_URL      = "https://cdn.norisk.gg/assets"
	MOJANG_SESSION_URL     = "https://sessionserver.mojang.com"

	DEFAULT_PACK = "norisk-prod"
	DEFAULT_MOD_DIR = "mods/NoRiskClient"

	MOD_INDEX = ".nrc-mod-index.json"
	ASSET_INDEX = ".nrc-asset-index.json"

	// Launchers
	PRISM_DIR = "PrismLauncher"
	PRISM_FLATPAK = "org.prismlauncher.PrismLauncher"
	PRISM_CLASS = "org.prismlauncher.EntryPoint"

	MODRINTH_DIR = "ModrinthApp"
	MODRINTH_FLATPAK = "com.modrinth.ModrinthApp"
	MODRINTH_CLASS = "com.modrinth.theseus.MinecraftLaunch"
)
