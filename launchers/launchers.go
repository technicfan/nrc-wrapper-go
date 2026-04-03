package launchers

type Minecraft struct {
	profile       string
	version       string
	loader        string
	loader_version string
	username      string
	uuid          string
	token         string
}

func NewMinecraft(
	instance Instance,
) Minecraft {
	return Minecraft{
		instance.Name(),
		instance.Version(),
		instance.Loader(),
		instance.LoaderVersion(),
		"", "", "",
	}
}

func (minecraft Minecraft) Profile() string {
	return minecraft.profile
}

func (minecraft Minecraft) Version() string {
	return minecraft.version
}

func (minecraft Minecraft) Loader() string {
	return minecraft.loader
}

func (minecraft Minecraft) LoaderVersion() string {
	return minecraft.loader_version
}

func (minecraft Minecraft) Username() string {
	return minecraft.username
}

func (minecraft Minecraft) Uuid() string {
	return minecraft.uuid
}

func (minecraft Minecraft) Token() string {
	return minecraft.token
}

type Launcher interface {
	Id() string
	Name() string
	Dir() string
	Container() string
	InstanceDir() string
	GetCurrentInstanceDetails() (Minecraft, error)
	GetInstances([]string, []string, string) ([]Instance, error)
	Exists() bool
	IsRunning() bool
}

type mutable_launcher interface {
	Launcher
	ReplaceNormal()
}

type launcher_data struct {
	name string
	path string
	instance_dir string
	flatpak_id string
	replaced bool
}

func (data launcher_data) Name() string {
	return data.name
}

func (data launcher_data) Dir() string {
	return data.path
}

func (data launcher_data) Container() string {
	if data.flatpak_id != "" {
		return "flatpak"
	} else {
		return ""
	}
}

func (data launcher_data) InstanceDir() string {
	return data.instance_dir
}

func (data *launcher_data) ReplaceNormal() {
	if data.flatpak_id != "" {
		data.replaced = true
	}
}
