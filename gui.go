package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var MAIN_PACKS []string

func gui() {
	MAIN_PACKS = []string{"norisk-prod", "norisk-bughunter"}

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

	a := app.NewWithID("nrc-wrapper-go")
	w := a.NewWindow("nrc-wrapper-go")
	w.Resize(fyne.NewSize(800, 500))
	w.CenterOnScreen()

	tabs := container.NewAppTabs()

	for _, l := range order {
		var open_configs []*Instance
		lstack := container.NewStack()
		cws := container.NewMultipleWindows()
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
						show_all := !slices.Contains(MAIN_PACKS, selected)

						all_packs_toggle := widget.NewCheck("Show all", func(b bool) {
							if !b {
								reference = MAIN_PACKS
								pack_select.SetOptions([]string{packs.Packs[reference[0]].Name, make_unique(packs.Packs[reference[1]].Name, 1)})
							} else {
								reference = temp_ref
								pack_select.SetOptions(options)
							}
						})
						all_packs_toggle.SetChecked(show_all)
						all_packs_toggle.OnChanged(show_all)

						if v, e := packs.Packs[instance.Config.NrcPack]; e {
							selected = make_unique(v.Name, slices.Index(reference, instance.Config.NrcPack))
						}
						pack_select.SetSelected(selected)

						loader_warn := widget.NewLabel("")
						loader_warn.Alignment = fyne.TextAlignCenter
						loader_warn_update := func() {
							selected := pack_select.SelectedIndex()
							if selected != -1 {
								if p, e := packs.Packs[reference[selected]]; e && cmp_mc_versions(instance.LoaderVersion, p.Loaders[instance.Loader]) == -1 {
									loader_warn.SetText(fmt.Sprintf(
										"Please update your %s%s loader to version %s to use this pack",
										strings.ToUpper(instance.Loader[:1]),
										instance.Loader[1:], p.Loaders[instance.Loader],
									))
									loader_warn.Show()
								} else {
									if !loader_warn.Hidden {
										cw.Resize(fyne.NewSize(0, 0))
									}
									loader_warn.Hide()
								}
							}
						}
						loader_warn_update()

						pack_select.OnChanged = func(s string) {
							loader_warn_update()
						}

						notify_toggle := widget.NewCheck("Send notifications", func(b bool) {})
						notify_toggle.SetChecked(instance.Config.Notify)

						neofd_toggle := widget.NewCheck("Disable crash on failed download", func(b bool) {})
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
						save_button := widget.NewButton("Save", func() {
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
							layout.NewVBoxLayout(),
							loader_warn,
							container.New(layout.NewHBoxLayout(), nrc_toggle, container.NewGridWrap(fyne.NewSize(260, 35), pack_select), all_packs_toggle),
							container.New(layout.NewHBoxLayout(), notify_toggle, layout.NewSpacer(), neofd_toggle),
							container.New(layout.NewGridLayout(2), cancel_button, save_button),
						)
						cw.SetContent(content)

						open_configs = append(open_configs, instance)
						cw.CloseIntercept = func() {
							index := slices.Index(open_configs, instance)
							open_configs = slices.Delete(open_configs, index, index+1)
							instance.NewConfig = instance.Config
							cw.Close()
							if len(open_configs) == 0 {
								cws.Hide()
							}
						}

						cw.Move(fyne.NewPos(w.Canvas().Size().Width/2-content.MinSize().Width/2, w.Canvas().Size().Height/2-content.MinSize().Height))
						cws.Show()
						cws.Add(cw)
					}
				}
			},
		)
		lstack.Add(list)
		lstack.Add(cws)
		cws.Hide()
		tabs.Append(container.NewTabItem(l, lstack))
	}

	if len(order) == 0 {
		label := widget.NewLabel("No compatible instances found\nPlease create a compatible instance in Modrinth App or Prism Launcher")
		label.Alignment = fyne.TextAlignCenter
		tabs.Append(container.NewTabItem("Nothing found", container.NewCenter(label)))
	}

	packs_main_first := MAIN_PACKS
	for id := range packs.Packs {
		if !slices.Contains(packs_main_first, id) {
			packs_main_first = append(packs_main_first, id)
		}
	}

	var packs_string_builder strings.Builder
	for _, id := range packs_main_first {
		pack := packs.Packs[id]
		var loaders []string
		for l, v := range pack.Loaders {
			loaders = append(loaders, fmt.Sprintf("%s%s \u2265 %s", strings.ToUpper(l[:1]), l[1:], v))
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
	scroll := container.NewVScroll(container.NewCenter(widget.NewRichTextFromMarkdown(packs_string_builder.String())))
	tabs.Append(container.NewTabItem("NRC packs", scroll))

	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.ShowAndRun()
}
