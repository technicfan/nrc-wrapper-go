package gui

import (
	"fmt"
	"log"
	"main/api"
	"main/fetcher"
	"main/globals"
	"main/launchers"
	"main/packs"
	"main/platform"
	"main/utils"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func Gui() {
	a := app.NewWithID("nrc-wrapper-go")
	w := a.NewWindow("nrc-wrapper-go")
	w.Resize(fyne.NewSize(800, 500))
	w.CenterOnScreen()

	running_launchers := platform.Get_running_launchers()

	if len(running_launchers) == 0 {
		ex, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		ex, err = filepath.EvalSymlinks(ex)
		if err != nil {
			log.Fatal(err)
		}
		if strings.Contains(ex, `\`) {
			ex = strings.ReplaceAll(ex, `\`, `\\`)
		}
		v, err := api.Get_norisk_versions()
		if err != nil {
			log.Fatal(err)
		}
		launchers_dirs, order := packs.Get_launcher_dirs()
		packs := v.Packs.To_meta_packs()
		versions := packs.Versions
		loaders := packs.Loaders
		instances := make(map[string][]launchers.Instance)
		for i, l := range order {
			inst, err := launchers.Get_instances(launchers_dirs[l][0], l, launchers_dirs[l][1], versions, loaders, ex)
			if err != nil {
				order = slices.Delete(order, i, i+1)
			} else {
				slices.SortFunc(inst, func(a launchers.Instance, b launchers.Instance) int {
					return strings.Compare(a.Name, b.Name)
				})
				instances[l] = inst
			}
		}
		var unique_main []string
		for i := range globals.MAIN_PACKS {
			name := packs.Packs[globals.MAIN_PACKS[i]].Name
			unique_main = append(unique_main, utils.Make_unique(name, i))
		}

		tabs := container.NewAppTabs()

		for _, l := range order {
			var open_configs []*launchers.Instance

			lstack := container.NewStack()
			cws := container.NewMultipleWindows()
			list := container.NewVBox()
			loading_bar := widget.NewProgressBarInfinite()
			for i := range instances[l] {
				instance := &instances[l][i]

				line := container.NewHBox()
				line.Add(widget.NewLabel(fmt.Sprintf(
					"%s - %s %s",
					instance.Name,
					strings.ToUpper(instance.Loader[:1]) + instance.Loader[1:],
					instance.Version,
				)))
				line.Add(layout.NewSpacer())

				main_button := widget.NewButton("Refresh NRC", func() {})
				main_button.OnTapped = func() {
					lstack.Add(loading_bar)
					fyne.Do(func() {
						var env []string
						if instance.FlatpakId != "" {
							env = append(env, "FLATPAK_ID=" + instance.FlatpakId)
						}
						for k, v := range instance.Env {
							env = append(env, k + "=" + v)
						}
						cmd := exec.Command(ex, "--refresh")
						cmd.Dir = instance.McRoot
						cmd.Env = env
						_, err := cmd.Output()
						if err != nil {
							utils.Notify(
								fmt.Sprintf("%s failed: %s", main_button.Text, err.Error()),
								false, true,
							)
						} else {
							main_button.SetText("Refresh NRC")
							utils.Notify("Finished!", false, true)
						}
						lstack.Remove(loading_bar)
					})
				}
				if !instance.Config.Nrc {
					main_button.Hide()
					main_button.SetText("Download NRC")
				}
				line.Add(main_button)

				line.Add(widget.NewButton("Options", func() {
					if !slices.Contains(open_configs, instance) {
						cw := container.NewInnerWindow(instance.Name, nil)

						options, reference, has_main_pack := packs.Get_compatible_packs(
							instance.Version, instance.Loader,
						)
						temp_ref := reference
						pack_select := widget.NewSelect(options, func(s string) {})
						selected := instance.Config.NrcPack
						if !has_main_pack {
							selected = options[0]
							instance.Config.NrcPack = selected
							instance.NewConfig.NrcPack = selected
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

						if v, e := packs.Packs[instance.Config.NrcPack]; e {
							selected = utils.Make_unique(
								v.Name, slices.Index(reference, instance.Config.NrcPack),
							)
						}
						pack_select.SetSelected(selected)

						warn_label := widget.NewLabel("")
						warn_label.Alignment = fyne.TextAlignCenter
						warn_label.Hide()
						loader_warn_update := func() {
							selected := pack_select.SelectedIndex()
							if selected != -1 {
								if p, e := packs.Packs[reference[selected]]; e && utils.Cmp_versions(
									instance.LoaderVersion, p.Loaders[instance.Loader],
								) < 0 {
									warn_label.SetText(fmt.Sprintf(
										"Please update your %s%s loader to version %s to use this pack",
										strings.ToUpper(instance.Loader[:1]),
										instance.Loader[1:], p.Loaders[instance.Loader],
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
						notify_toggle.SetChecked(instance.Config.Notify)

						neofd_toggle := widget.NewCheck(
							"Disable crash on failed download", func(b bool) {},
						)
						neofd_toggle.SetChecked(instance.Config.Neofd)

						nrc_toggle := widget.NewCheck("Enable NRC", func(b bool) {
							if b {
								if pack_select.SelectedIndex() == -1 {
									pack_select.SetSelectedIndex(0)
								}
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

						cancel_button := widget.NewButton("Cancel", func() {
							cw.CloseIntercept()
						})
						save_button := widget.NewButton("Save", func() {})
						save_button.OnTapped = func() {
							instance.NewConfig.Nrc = nrc_toggle.Checked
							instance.NewConfig.Notify = notify_toggle.Checked
							instance.NewConfig.Neofd = neofd_toggle.Checked
							pack := pack_select.SelectedIndex()
							if pack != -1 {
								instance.NewConfig.NrcPack = reference[pack]
							}
							err := instance.Save(ex)
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
							fyne.Do(func() {
								time.Sleep(time.Millisecond * 350)
								if err == nil {
									cw.CloseIntercept()
									if instance.Config.Nrc {
										if main_button.Hidden {
											main_button.Show()
										}
									} else {
										if !main_button.Hidden {
											main_button.Hide()
										}
									}
								}
							})
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
							instance.NewConfig = instance.Config
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
					mods, err := fetcher.Get_installed_mods(instance.McRoot, instance.Config.ModDir)
					var mods_to_toggle []string

					var mod_list *fyne.Container
					if err == nil && len(mods) != 0 {
						mod_list = container.NewVBox()
						mod_names := make(map[string]string)
						if pack, e := v.Packs[instance.Config.NrcPack]; e {
							mod_names = pack.Mods.Get_names(mods)
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
						if err != nil {
							log.Printf(
								"Getting installed mods for %s failed: %s",
								instance.Name, err.Error(),
							)
						}
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
									filepath.Join(instance.McRoot, mods[id].Path()),
									filepath.Join(instance.McRoot, filepath.Dir(mods[id].Path()), new_name),
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
			tabs.Append(container.NewTabItem(l, lstack))
		}

		if len(order) == 0 {
			label := widget.NewLabel(
				"No compatible instances found\nPlease create a compatible instance in Modrinth App or Prism Launcher",
			)
			label.Alignment = fyne.TextAlignCenter
			tabs.Append(container.NewTabItem("Nothing found", container.NewCenter(label)))
		}

		var packs_main_first []string
		for id := range packs.Packs {
			if !slices.Contains(globals.MAIN_PACKS, id) {
				packs_main_first = append(packs_main_first, id)
			}
		}
		slices.Sort(packs_main_first)
		packs_main_first = append(globals.MAIN_PACKS, packs_main_first...)

		var packs_string_builder strings.Builder
		for _, id := range packs_main_first {
			pack := packs.Packs[id]
			var loaders []string
			for l, v := range pack.Loaders {
				var version string
				if v != "0" {
					version = fmt.Sprintf(" \u2265 %s", v)
				}
				loaders = append(
					loaders,
					fmt.Sprintf("%s%s%s", strings.ToUpper(l[:1]), l[1:], version),
				)
			}
			fmt.Fprintf(&packs_string_builder, `
## %s (%s)

- %s

- Supported Minecraft Versions: %s

- Supported Modloaders: %s
---`,
				pack.Name, id, pack.Desc, strings.Join(pack.Versions, ", "), strings.Join(loaders, ", "),
			)
		}
		scroll := container.NewVScroll(
			container.NewCenter(widget.NewRichTextFromMarkdown(packs_string_builder.String())),
		)
		tabs.Append(container.NewTabItem("NRC packs", scroll))

		tabs.SetTabLocation(container.TabLocationTop)

		w.SetContent(tabs)
	} else {
		error_msg := widget.NewLabel(
			fmt.Sprintf("These Launchers are currently running on your system:\n%s\nPlease close them first to use this app.",
				strings.Join(running_launchers, "\n"),
			),
		)
		error_msg.Alignment = fyne.TextAlignCenter
		error_dialog := dialog.NewCustom("Error", "Close", error_msg, w)
		error_dialog.SetOnClosed(func() {
			w.Close()
		})
		error_dialog.Show()
	}

	w.ShowAndRun()
}
