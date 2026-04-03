package gui

import (
	"fmt"
	"log"
	"main/api"
	"main/config"
	"main/fetcher"
	"main/globals"
	"main/launchers"
	"main/packs"
	"main/utils"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func addInstances(
	l launchers.Launcher,
	versions []string,
	loaders []string,
	unique_main []string,
	packs packs.MetaPacks,
	v api.Versions,
	lstack *fyne.Container,
	info_box *fyne.Container,
	w fyne.Window,
	ex string,
) bool {
	instances, err := l.GetInstances(versions, loaders, ex)
	if (len(instances) == 0) {
		heading := info_box.Objects[0].(*fyne.Container).Objects[0].(*widget.RichText)
		desc := info_box.Objects[1].(*widget.Label)
		if err != nil {
			heading.ParseMarkdown("## An error occurred while getting instances")
			desc.SetText(err.Error())
		} else {
			heading.ParseMarkdown("## No compatible instances found")
			desc.SetText("See the \"NRC Packs\" tabs for compatibility info")
		}
		lstack.Add(container.NewCenter(info_box))
		return false
	}
	var open_configs []launchers.Instance
	cws := container.NewMultipleWindows()
	list := container.NewVBox()
	for i := range instances {
		instance := instances[i]

		line := container.NewHBox()
		line.Add(widget.NewLabel(fmt.Sprintf(
			"%s - %s %s",
			instance.Name(),
			strings.ToUpper(instance.Loader()[:1]) + instance.Loader()[1:],
			instance.Version(),
		)))
		line.Add(layout.NewSpacer())

		main_button := widget.NewButton("Refresh NRC", func() {})
		main_button.OnTapped = func() {
			text := main_button.Text
			main_button.SetText("Loading...")
			main_button.Disable()
			go func ()  {
				err = fetcher.Fetch(v, config.NewConfigFromGui(l, instance))
				if err != nil {
					fyne.Do(func() {
						main_button.SetText("Failed")
				    })
					log.Printf("%s failed: %s\n", text, err.Error())
				} else {
					fyne.Do(func() {
						main_button.SetText("Finished")
				    })
				}
				fyne.Do(func() {
					main_button.Enable()
				})
				time.Sleep(time.Second)
				fyne.Do(func() {
					main_button.SetText("Refresh NRC")
				})
			}()
		}
		if !instance.Nrc() {
			main_button.Hide()
			main_button.SetText("Download NRC")
		}
		line.Add(main_button)

		line.Add(widget.NewButton("Options", func() {
			if !slices.Contains(open_configs, instance) {
				cw := container.NewInnerWindow(instance.Name(), nil)

				options, reference, has_main_pack := packs.CompatiblePacks(
					instance.Version(), instance.Loader(),
				)
				temp_ref := reference
				pack_select := widget.NewSelect(options, func(s string) {})
				selected := instance.Pack()
				if !has_main_pack {
					selected = options[0]
					instance.FixPack(selected)
				}
				show_all := !slices.Contains(globals.MAIN_PACKS, selected)

				all_packs_toggle := widget.NewCheck("Show all", func(b bool) {
					if !b {
						var new_reference, new_options []string
						for i := range globals.MAIN_PACKS {
							if slices.Contains(reference, globals.MAIN_PACKS[i]) {
								new_reference = append(new_reference, globals.MAIN_PACKS[i])
								new_options = append(new_options, unique_main[i])
							}
						}
						reference = new_reference
						pack_select.SetOptions(new_options)
					} else {
						reference = temp_ref
						pack_select.SetOptions(options)
					}
				})
				all_packs_toggle.SetChecked(show_all)
				all_packs_toggle.OnChanged(show_all)
				if !has_main_pack {
					all_packs_toggle.Disable()
				}

				if v, e := packs.Packs[instance.Pack()]; e {
					selected = utils.Unique(v.Name, slices.Index(reference, instance.Pack()))
				}
				pack_select.SetSelected(selected)

				warn_label := widget.NewLabel("")
				warn_label.Alignment = fyne.TextAlignCenter
				warn_label.Hide()
				loader_warn_update := func() {
					selected := pack_select.SelectedIndex()
					if selected != -1 {
						if p, e := packs.Packs[reference[selected]]; e && utils.CmpVersions(
							instance.LoaderVersion(), p.Loaders[instance.Loader()],
						) < 0 {
							warn_label.SetText(fmt.Sprintf(
								"Please update your %s%s loader to version %s to use this pack",
								strings.ToUpper(instance.Loader()[:1]),
								instance.Loader()[1:], p.Loaders[instance.Loader()],
							))
							warn_label.Show()
						} else {
							warn_label.Hide()
							if !warn_label.Hidden {
								cw.Resize(fyne.NewSize(0, 0))
							}
						}
					}
				}
				loader_warn_update()

				pack_select.OnChanged = func(s string) {
					loader_warn_update()
				}

				notify_toggle := widget.NewCheck("Send notifications", func(b bool) {})
				notify_toggle.SetChecked(instance.Notify())

				neofd_toggle := widget.NewCheck(
					"Disable crash on failed download", func(b bool) {},
				)
				neofd_toggle.SetChecked(instance.Neofd())

				nrc_toggle := widget.NewCheck("Enable NRC", func(b bool) {
					if b {
						if pack_select.SelectedIndex() == -1 {
							pack_select.SetSelectedIndex(0)
						}
						pack_select.Enable()
						if has_main_pack {
							all_packs_toggle.Enable()
						}
						notify_toggle.Enable()
						neofd_toggle.Enable()
					} else {
						pack_select.Disable()
						notify_toggle.Disable()
						neofd_toggle.Disable()
					}
				})
				nrc_toggle.SetChecked(instance.Nrc())
				if !instance.Nrc() {
					pack_select.Disable()
					all_packs_toggle.Disable()
					notify_toggle.Disable()
					neofd_toggle.Disable()
				}

				cancel_button := widget.NewButton("Cancel", func() {
					cw.CloseIntercept()
				})
				save_button := widget.NewButton("Save", func() {})
				save_button.OnTapped = func() {
					var pack string
					pack_index := pack_select.SelectedIndex()
					if pack_index != -1 {
						pack = reference[pack_index]
					}
					err := instance.Save(
						nrc_toggle.Checked,
						notify_toggle.Checked,
						neofd_toggle.Checked,
						pack,
						ex,
					)
					if err != nil {
						warn_label.SetText(
							"An error occurred while saving\nSee log (stdout) for more details",
						)
						log.Printf("Failed to save %s: %s", instance.Name, err.Error())
						warn_label.Show()
					} else {
						warn_label.SetText("Your settings have been saved successfully")
						warn_label.Show()
					}
					if err == nil {
						go func ()  {
							time.Sleep(time.Millisecond * 350)
							fyne.Do(func () {
								cw.CloseIntercept()
								if instance.Nrc() {
									if main_button.Hidden {
										main_button.Show()
									}
								} else {
									if !main_button.Hidden {
										main_button.Hide()
									}
								}
							})
						}()
					}
				}

				content := container.New(
					layout.NewVBoxLayout(),
					warn_label,
					container.New(
						layout.NewHBoxLayout(),
						nrc_toggle,
						container.NewGridWrap(fyne.NewSize(260, 35), pack_select),
						all_packs_toggle,
					),
					container.New(
						layout.NewHBoxLayout(),
						notify_toggle,
						layout.NewSpacer(),
						neofd_toggle,
					),
					container.New(layout.NewGridLayout(2), cancel_button, save_button),
				)
				cw.SetContent(content)

				cw.CloseIntercept = func() {
					index := slices.Index(open_configs, instance)
					open_configs = slices.Delete(open_configs, index, index+1)
					cw.Close()
					for i := range cws.Windows {
						if cws.Windows[i] == cw {
							cws.Windows = slices.Delete(cws.Windows, i, i + 1)
							break
						}
					}
					if len(open_configs) == 0 {
						cws.Hide()
					}
				}

				cw.Move(
					fyne.NewPos(w.Canvas().Size().Width/2-content.MinSize().Width/2,
						w.Canvas().Size().Height/2-content.MinSize().Height,
					),
				)
				cws.Show()

				open_configs = append(open_configs, instance)
				cws.Add(cw)
			}
		}))
		line.Add(widget.NewButton("NRC Mods", func() {
			mods, _ := fetcher.GetInstalledMods(instance.Path(), instance.ModDir())
			var mods_to_toggle []string

			var mod_list *fyne.Container
			if len(mods) != 0 {
				mod_list = container.NewVBox()
				mod_names := make(map[string]string)
				if pack, e := v.Packs[instance.Pack()]; e {
					mod_names = pack.Mods.DisplayNames(mods)
				}
				for _, id := range slices.Sorted(maps.Keys(mods)) {
					line := container.NewGridWithColumns(2)
					var name string
					if n, e := mod_names[id]; e {
						name = n
					} else {
						name = id
					}
					line.Add(widget.NewLabel(name))
					toggle := widget.NewButton("", func() {})
					if mods[id].Enabled() {
						toggle.SetText("Disable")
						toggle.SetIcon(theme.CancelIcon())
						toggle.Importance = widget.DangerImportance
					} else {
						toggle.SetText("Enable")
						toggle.SetIcon(theme.ContentAddIcon())
						toggle.Importance = widget.HighImportance
					}
					toggle.Refresh()
					toggle.OnTapped = func() {
						if !slices.Contains(mods_to_toggle, id) {
							mods_to_toggle = append(mods_to_toggle, id)
						}
						if toggle.Text == "Disable" {
							toggle.SetText("Enable")
							toggle.SetIcon(theme.ContentAddIcon())
							toggle.Importance = widget.HighImportance
						} else {
							toggle.SetText("Disable")
							toggle.SetIcon(theme.CancelIcon())
							toggle.Importance = widget.DangerImportance
						}
						toggle.Refresh()
					}
					line.Add(container.NewGridWithColumns(2, layout.NewSpacer(), toggle))
					mod_list.Add(line)
					mod_list.Add(widget.NewSeparator())
				}
			} else {
				mod_list = container.NewCenter(widget.NewLabel("No Noriskclient mods found"))
			}

			content := container.NewStack()
			content.Add(container.New(
				layout.NewCustomPaddedLayout(40, 0, 0, 0),
				container.NewVScroll(container.New(
					layout.NewCustomPaddedLayout(0, 0, 100, 100),
					mod_list,
				)),
			))

			close_view := func () {
				lstack.Remove(content)
				lstack.Objects[0].Show()
				if len(lstack.Objects[1].(*container.MultipleWindows).Windows) != 0 {
					lstack.Objects[1].Show()
				}
			}

			content.Add(container.NewVBox(container.NewGridWithColumns(2,
				widget.NewButton("Back", close_view),
				widget.NewButton("Save", func() {
					for _, id := range mods_to_toggle {
						var new_name string
						if mods[id].Enabled() {
							new_name = mods[id].Filename() + ".disabled"
						} else {
							new_name = strings.TrimSuffix(mods[id].Filename(), ".disabled")
						}
						err := os.Rename(
							filepath.Join(instance.Path(), mods[id].Path()),
							filepath.Join(instance.Path(), filepath.Dir(mods[id].Path()), new_name),
						)
						if err != nil {
							log.Printf(
								"Failed to toggle %s: %s",
								mods[id].Filename(), err.Error(),
							)
						}
					}
					close_view()
				}),
			)))
			for i := range lstack.Objects {
				lstack.Objects[i].Hide()
			}
			lstack.Add(content)
		}))
		line.Add(widget.NewLabel(""))
		list.Add(line)
		list.Add(widget.NewSeparator())
	}
	lstack.Add(container.NewVScroll(
		container.New(layout.NewCustomPaddedLayout(2.5, 2.5, 25, 25), list),
	))
	lstack.Add(cws)
	cws.Hide()
	return true
}
