package gui

import (
	"fmt"
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

	loading := widget.NewLabel("Loading...")
	w.SetContent(container.NewCenter(container.NewVBox(
		loading,
		widget.NewProgressBarInfinite(),
	)))

	set_error_text := func (text string)  {
		fyne.Do(func() {
			loading.SetText("Error: " + text)
		})
	}

	go func() {
		var api_endpoint string
		if os.Getenv("STAGING") != "" {
			api_endpoint = globals.STAGING_NORISK_API_ENDPOINT
		} else {
			api_endpoint = globals.NORISK_API_ENDPOINT
		}

		ex, err := os.Executable()
		if err != nil {
			set_error_text(err.Error())
		}
		ex, err = filepath.EvalSymlinks(ex)
		if err != nil {
			set_error_text(err.Error())
		}
		if strings.Contains(ex, `\`) {
			ex = strings.ReplaceAll(ex, `\`, `\\`)
		}
		v, err := api.GetVersions(api_endpoint)
		if err != nil {
			set_error_text(err.Error())
		}
		packs := v.Packs.MetaPacks()
		var unique_main []string
		for i := range globals.MAIN_PACKS {
			name := packs.Packs[globals.MAIN_PACKS[i]].Name
			unique_main = append(unique_main, utils.Unique(name, i))
		}
		launchers, err := launchers.GetLaunchers()

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
			for _, l := range launchers {
				lstack := container.NewStack()
				heading := widget.NewRichTextFromMarkdown(fmt.Sprintf("## %s is currently running", l.Name()))
				desc := widget.NewLabel("Close it before changing anything here to prevent corruption")
				desc.Alignment = fyne.TextAlignCenter
				info_box := container.NewVBox(
					container.NewCenter(heading),
					desc,
				)
				if !l.IsRunning() {
					if !addInstances(
						l, unique_main,
						packs,
						v, lstack,
						info_box,
						w, ex,
					) {
						continue
					}
				} else {
					running_box := container.NewCenter()
					remove_running_box := func () {
						lstack.Remove(running_box)
						addInstances(
							l, unique_main,
							packs,
							v, lstack,
							info_box,
							w, ex,
						)
					}
					refresh := func () {
						if l.IsRunning() {
							heading.ParseMarkdown(fmt.Sprintf("## %s is still running", l.Name()))
						} else {
							remove_running_box()
						}
					}
					running_box.Add(container.NewVBox(
						info_box,
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
			var lines []string
			for l, v := range pack.Support {
				var version string
				if v.LoaderVersion != "0" {
					version = fmt.Sprintf(" \u2265 %s", v.LoaderVersion)
				}
				line := fmt.Sprintf(`
- %s%s%s
    - %s
`, strings.ToUpper(l[:1]), l[1:], version, strings.Join(v.Versions, ", "))
				lines = append(lines, line)
			}
			fmt.Fprintf(&packs_string_builder, `
## %s (%s)

- %s

Support:

%s
---`,
				pack.Name, id, pack.Desc, strings.Join(lines, "\n"),
			)
		}
		scroll := container.NewVScroll(
			container.NewCenter(widget.NewRichTextFromMarkdown(packs_string_builder.String())),
		)
		tabs.Append(container.NewTabItem("NRC packs", scroll))

		tabs.SetTabLocation(container.TabLocationTop)

		fyne.Do(func() {
			w.SetContent(tabs)
		})
	}()

	w.ShowAndRun()
}
