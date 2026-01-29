### nrc-prism-wrapper-go

---

This was a small challenge for me.<br>
I wanted to write something in a language that was new for me and I decided for Go and to translate/rewrite [nrc-prism-wrapper](https://github.com/ThatCuteOne/nrc-prism-wrapper) in Go.<br>
[ThatCuteOne](https://github.com/ThatCuteOne) is still maintaining their project btw, so you might want to check it out too.

---

It can run through Prism Launcher and Modrinth Launcher on Windows and Linux! (idk about mac)<br>
Once again most algorithms are **NOT** my own but those by [ThatCuteOne](https://github.com/ThatCuteOne), but everything after v1.0 and [`types.go`](./types.go) is my work.

---

It installs the mods in a seperate directory from all your other mods to prevent the launcher messing with them (only supported with fabric).<br>
It also checks if the correct loader and loader version is installed to run the selected nrc modpack.

---

### Config:

The wrapper can be configured through environment variables

|               Variable                |                                                        Description                                                         |
| :-----------------------------------: | :------------------------------------------------------------------------------------------------------------------------: |
|              `NRC_PACK`               | This lets you choose, which nrc modpack to use; You can see available option when using the `--packs` flag in the terminal |
|             `NRC_MOD_DIR`             |        This lets you change the directory the wrapper installs the nrc mods in (the default is `mods/NoRiskClient`)        |
|              `LAUNCHER`               | The wrapper generally detects the launcher by itself, but in case it doesn't you have the option for `prism` or `modrinth` |
|              `PRISM_DIR`              |                         The data directory of Prism Launcher in case it's not the default location                         |
|            `MODRINTH_DIR`             |                       The data directory of Modrinth Launcher in case it's not the default location                        |
|               `NOTIFY`                |      Set it to `true\|True\|1` or `false\|False\|0` to enable/disable notifications (enabled by default for modrinth)      |
| `NO_ERROR_ON_FAILED_DOWNLOAD`/`NEOFD` |                              Set it to anything to stop crashing if a file fails downloading                               |

**Info:**

~~The `--packs` flag only works on Linux as on Windows an app is either CLI or GUI. I chose GUI because otherwise when starting Minecraft with the wrapper, a CMD window would always appear.~~<br>
Managed to fix this with a weird solution (I don't know much about Windows).

---

To build it yourself simply install Go and run

On Linux:

```
go build -o nrc-wrapper-go
```

or for Windows build on Linux (how I built the releases):

- install mingw-w64-gcc

```
GOOS=windows GOARCH=386 CGO_ENABLED=1 CXX=i686-w64-mingw32-g++ CC=i686-w64-mingw32-gcc go build -ldflags -H=windowsgui -o nrc-wrapper-go.exe
```

And on Windows:

- download mingw64 and add the `bin` directory to your path

```
go env -w CGO_ENABLED=1
go build -ldflags -H=windowsgui -o nrc-wrapper-go.exe
```

in the git repository.

The binary that you'll have now has to be added as the wrapper command of a supported Minecraft (check `--packs` for a list) instance in a supported Launcher.<br>
Just launch and enjoy.

---

Things to keep in mind:

- this is not official and not affiliated with NoRiskClient
    - if issues occur first look if this project is the cause
- first launch takes longer as it needs to download everything
- when running in Flatpak, you'll need to place the wrapper in a directory the app has full access to
- ~~on Windows there is no way to replace the current process, so minecraft will just continue running if you terminate the instance from your launcher~~
    - managed to fix it with winjobs using a 5yo go package (i don't really know, what this is though :))
