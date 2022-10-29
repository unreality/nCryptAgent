tools\x86_64-w64-mingw32uwp-windres.exe -i resources.rc -o rsrc.syso -O coff
go build -ldflags "-H=windowsgui" -o build\ncryptagent.exe