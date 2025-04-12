.PHONY: all mac win

all: mac win

mac:
	@echo "Building for macOS..."
	go build -o maschine_chords_converter ./cmd

win:
	@echo "Building for Windows..."
	env GOOS=windows GOARCH=amd64 go build -o maschine_chords_converter.exe ./cmd
