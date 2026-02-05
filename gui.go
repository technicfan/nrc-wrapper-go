package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func gui() {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	ex, err = filepath.EvalSymlinks(ex)
	if err != nil {
		log.Fatal(err)
	}
	v, err := get_norisk_versions(NORISK_API_URL)
	if err != nil {
		log.Fatal(err)
	}
	packs := v.Packs.to_meta_packs()
	versions := packs.Versions
	loaders := packs.Loaders
	launchers := get_launcher_dirs()
	instances := make(map[string][]Instance)
	for l := range launchers {
		i, err := get_instances(launchers[l][0], launchers[l][1], versions, loaders, ex)
		if err != nil {
			log.Fatal(err)
		}
		instances[l] = i
	}

	a := app.New()
	w := a.NewWindow("nrc-wrapper-go")

	tabs := container.NewAppTabs()

	var open_configs []*Instance

	for l := range instances {
		list := widget.NewList(
			func() int {
				return len(instances[l])
			},
			func() fyne.CanvasObject {
				return widget.NewButton("Button", func() {})
			},
			func(i widget.ListItemID, o fyne.CanvasObject) {
				instance := &instances[l][i]
				o.(*widget.Button).SetText(fmt.Sprintf(
					"%s - %s %s", instance.Name, instance.Loader, instance.Version,
				))
				o.(*widget.Button).OnTapped = func () {
					if !slices.Contains(open_configs, instance) {
						cw := a.NewWindow(instance.Name)

						pack_select := widget.NewSelect(packs.get_compatible_packs(instance.Version, instance.Loader), func(s string) {
							instance.NewConfig.NrcPack = s
						})
						pack_select.SetSelected(instance.Config.NrcPack)

						notify_toggle := widget.NewCheck("Send notifications", func(b bool) {
							instance.NewConfig.Notify = b
						})
						notify_toggle.SetChecked(instance.Config.Notify)

						neofd_toggle := widget.NewCheck("Disable crash on failed download", func(b bool) {
							instance.NewConfig.Neofd = b
						})
						neofd_toggle.SetChecked(instance.Config.Neofd)

						nrc_toggle := widget.NewCheck("Enable NRC", func(b bool) {
							instance.NewConfig.Nrc = b
							if b {
								pack_select.Enable()
								notify_toggle.Enable()
								neofd_toggle.Enable()
							} else {
								pack_select.Disable()
								notify_toggle.Disable()
								neofd_toggle.Disable()
							}
						})
						nrc_toggle.SetChecked(instance.Config.Nrc)
						if !nrc_toggle.Checked {
							pack_select.Disable()
							notify_toggle.Disable()
							neofd_toggle.Disable()
						}

						cancel_botton := widget.NewButton("Cancel", func() {
							cw.Close()
						})
						save_botton := widget.NewButton("Save", func() {
							instance.update(ex)
							cw.Close()
						})

						content := container.New(
							layout.NewGridLayout(1),
							nrc_toggle,
							pack_select,
							notify_toggle,
							neofd_toggle,
							cancel_botton,
							save_botton,
						)

						cw.SetContent(content)
						open_configs = append(open_configs, instance)
						cw.SetOnClosed(func ()  {
							index := slices.Index(open_configs, instance)
							open_configs = slices.Delete(open_configs, index, index + 1)
							instance.NewConfig = instance.Config
						})
						cw.Show()
					}
				}
			})
		content := container.NewTabItem(l, list)
		tabs.Append(content)
	}

	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.ShowAndRun()
}
