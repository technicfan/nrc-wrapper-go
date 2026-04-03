package globals

var MAIN_PACKS = []string{"norisk-prod", "norisk-bughunter", "norisk-development"}

const (
	NORISK_API_ENDPOINT         = "https://api.norisk.gg/api/v1"
	STAGING_NORISK_API_ENDPOINT = "https://api-staging.norisk.gg/api/v1"
	NORISK_ASSETS_ENDPOINT      = "https://cdn.norisk.gg/assets"
	MOJANG_SESSION_ENDPOINT     = "https://sessionserver.mojang.com"

	DEFAULT_PACK    = "norisk-prod"
	DEFAULT_MOD_DIR = "mods/NoRiskClient"

	MOD_INDEX   = ".nrc-mod-index.json"
	ASSET_INDEX = ".nrc-asset-index.json"
	TOKEN_STORE = "norisk_data.json"
)
