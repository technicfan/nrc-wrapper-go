package gui

import (
	"fmt"
	"log"
	"main/api"
	"main/globals"
	"main/launchers"
	"main/utils"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func Gui() {
	a := app.NewWithID("nrc-wrapper-go")
	w := a.NewWindow("nrc-wrapper-go")
	w.Resize(fyne.NewSize(800, 500))
	w.CenterOnScreen()

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
	v, err := api.GetVersions()
	if err != nil {
		log.Fatal(err)
	}
	packs := v.Packs.MetaPacks()
	var unique_main []string
	for i := range globals.MAIN_PACKS {
		name := packs.Packs[globals.MAIN_PACKS[i]].Name
		unique_main = append(unique_main, utils.Unique(name, i))
	}
	instances, order, err := launchers.GetInstances(packs.Versions, packs.Loaders, ex)

	tabs := container.NewAppTabs()

	if err != nil {
		tabs.Append(container.NewTabItem(
			"Nothing found",
			container.NewCenter(container.NewVBox(
				container.NewCenter(widget.NewRichTextFromMarkdown(`## No compatible instances found`)),
				widget.NewLabel("Create a compatible instance in Modrinth App or Prism Launcher"),
			)),
		))
	} else {
		for _, l := range order {
			lstack := container.NewStack()
			if inst, e := instances[l.Name()]; e {
				addInstances(inst, l, unique_main, packs, v, lstack, w, ex)
			} else {
				delete(instances, l.Name())
				running_box := container.NewCenter()
				heading := widget.NewRichTextFromMarkdown(`## This launcher is currently running`)
				info := container.NewVBox(
					container.NewCenter(heading),
					widget.NewLabel("Close it before changing anything here to prevent corruption"),
				)
				remove_running_box := func () {
					inst, err := l.GetInstances(packs.Versions, packs.Loaders, ex)
					lstack.Remove(running_box)
					if err == nil && len(inst) > 0 {
						addInstances(inst, l, unique_main, packs, v, lstack, w, ex)
					} else {
						lstack.Add(container.NewCenter(
							widget.NewRichTextFromMarkdown(`## No compatible instances found for this launcher`),
						))
					}
				}
				refresh := func () {
					if l.IsRunning() {
						heading.ParseMarkdown(`## This launcher is still running`)
					} else {
						remove_running_box()
					}
				}
				running_box.Add(container.NewVBox(
					info,
					container.NewGridWithColumns(2,
						widget.NewButton("Refresh", refresh),
						widget.NewButton("Ignore", remove_running_box),
					),
				))
				lstack.Add(running_box)
			}
			tabs.Append(container.NewTabItem(l.Name(), lstack))
		}
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

	w.ShowAndRun()
}
