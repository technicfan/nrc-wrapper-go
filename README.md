### nrc-prism-wrapper-go

---

This was a small challenge for me.
I wanted to write something in a language that was new for me and I decided for Go and to translate/rewrite [nrc-prism-wrapper](https://github.com/ThatCuteOne/nrc-prism-wrapper) in Go.

---

Currently it can only launch the NoRiskClient through the PrismLauncher on Windows and Linux (idk about mac).
Maybe I will try to add more launchers in the future.
Once again most algorithms are **NOT** my own but those by [ThatCuteOne](https://github.com/ThatCuteOne)

---

To build it yourself simply install Go and run

```
go build
```

in the git repository.

The binary that you'll have now has to be added as the wrapper command of a Minecraft 1.21-1.21.8 Fabric instance in PrismLauncher.<br>
Just launch and enjoy.

---

You can chose the NRC Pack via the environment variable `NRC_PACK` (choose between `norisk-prod`, `norisk-dev` or `norisk-bughunter`)
