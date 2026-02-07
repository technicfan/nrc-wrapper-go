package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type richString struct {
	string

	Id string
}

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
	launchers, order := get_launcher_dirs()
	instances := make(map[string][]Instance)
	for _, l := range order {
		i, err := get_instances(launchers[l][0], launchers[l][1], versions, loaders, ex)
		if err != nil {
			log.Fatal(err)
		}
		instances[l] = i
	}

	var open_configs []*Instance

	a := app.New()
	w := a.NewWindow("nrc-wrapper-go")

	tabs := container.NewAppTabs()
	stack := container.NewStack()

	for _, l := range order {
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
				o.(*widget.Button).OnTapped = func() {
					if !slices.Contains(open_configs, instance) {
						cw := container.NewInnerWindow(instance.Name, nil)

						options, reference := packs.get_compatible_packs(instance.Version, instance.Loader)
						temp_ref := reference
						pack_select := widget.NewSelect(options, func(s string) {})
						selected := instance.Config.NrcPack
						show_all := !regexp.MustCompile("norisk-prod|norisk-bughunter|^$").MatchString(selected)

						if v, e := packs.Packs[instance.Config.NrcPack]; e {
							selected = make_unique(v.Name, slices.Index(reference, instance.Config.NrcPack))
						}
						pack_select.SetSelected(selected)

						all_packs_toggle := widget.NewCheck("Show all", func(b bool) {
							if !b {
								reference = []string{"norisk-prod", "norisk-bughunter"}
								pack_select.SetOptions([]string{packs.Packs[reference[0]].Name, make_unique(packs.Packs[reference[1]].Name, 1)})
							} else {
								reference = temp_ref
								pack_select.SetOptions(options)
							}
						})
						all_packs_toggle.SetChecked(show_all)
						if !show_all {
							reference = []string{"norisk-prod", "norisk-bughunter"}
							pack_select.SetOptions([]string{packs.Packs[reference[0]].Name, packs.Packs[reference[1]].Name})
						}

						notify_toggle := widget.NewCheck("Send notifications", func(b bool) {})
						notify_toggle.SetChecked(instance.Config.Notify)

						neofd_toggle := widget.NewCheck("Disable crash on failed download", func(b bool) {})
						neofd_toggle.SetChecked(instance.Config.Neofd)

						nrc_toggle := widget.NewCheck("Enable NRC", func(b bool) {
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
						if !instance.Config.Nrc {
							pack_select.Disable()
							notify_toggle.Disable()
							neofd_toggle.Disable()
						}

						cancel_botton := widget.NewButton("Cancel", func() {
							cw.CloseIntercept()
						})
						save_botton := widget.NewButton("Save", func() {
							instance.NewConfig.Nrc = nrc_toggle.Checked
							instance.NewConfig.Notify = notify_toggle.Checked
							instance.NewConfig.Neofd = neofd_toggle.Checked
							pack := pack_select.SelectedIndex()
							if pack != -1 {
								instance.NewConfig.NrcPack = reference[pack]
							}

							instance.save(ex)
							cw.CloseIntercept()
						})

						content := container.New(
							layout.NewFormLayout(),
							nrc_toggle,
							container.New(layout.NewFormLayout(), pack_select, all_packs_toggle),
							notify_toggle,
							neofd_toggle,
							cancel_botton,
							save_botton,
						)
						cw.SetContent(content)

						cw.OnDragged = func(de *fyne.DragEvent) {
							cw.Move(de.AbsolutePosition.SubtractXY(cw.MinSize().Width / 2, 10))
						}

						open_configs = append(open_configs, instance)
						cw.CloseIntercept = func() {
							index := slices.Index(open_configs, instance)
							open_configs = slices.Delete(open_configs, index, index+1)
							instance.NewConfig = instance.Config
							cw.Close()
						}

						center := container.New(layout.NewCenterLayout(), cw)
						stack.Add(center)
					}
				}
			},
		)
		content := container.NewTabItem(l, list)
		tabs.Append(content)
	}

	tabs.SetTabLocation(container.TabLocationTop)

	stack.Add(tabs)

	w.SetContent(stack)
	w.ShowAndRun()
}
