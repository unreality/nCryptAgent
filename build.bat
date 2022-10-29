windres.exe -i resources.rc -o rsrc.syso -O coff
go build -ldflags "-H=windowsgui" -o build\nCryptAgent.exe