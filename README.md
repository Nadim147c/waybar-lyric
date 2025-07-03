# waybar-lyric

A CLI tool that displays lyrics on your
[Waybar](https://github.com/Alexays/Waybar) for Linux systems.

<video src="https://github.com/user-attachments/assets/a352dc99-8736-4c34-84f3-7492c607c7f0" height="360" controls></video>

## Description

`waybar-lyric` fetches and displays real-time lyrics on your Waybar. It provides a
scrolling lyrics display that syncs with your currently playing music, enhancing your
desktop music experience.

## Features

- Real-time display of the current song's lyrics
- Click to toggle play/pause
- Smart caching system:
  - Stores available lyrics locally to reduce API requests
  - Remembers songs without lyrics to prevent unnecessary API calls
- Custom waybar tooltip
- Configurable maximum text length
- Detailed logging options
- Profanity filter
  - Partial (`badword` -> `b*****d`)
  - Full (`badword` -> `*******`)

## Installation

### Prerequisites

- [Waybar](https://github.com/Alexays/Waybar)
- A working Spotify installation
- DBus connectivity

### Install

#### Prerequisites

- [go](https://go.dev/)

#### Installation

- From [AUR](https://aur.archlinux.org/packages); Recommended for Arch `btw` users.

```bash
yay -S waybar-lyric-git
```

- Or from [Nixpkgs](https://github.com/NixOS/nixpkgs)

On NixOS:

```nix
  environment.systemPackages = [
    pkgs.waybar-lyric
  ];
```

On Non NixOS:

```bash
# without flakes:
nix-env -iA nixpkgs.waybar-lyric
```

- With `go install`
  > Note: You have to make sure that `$GOPATH/bin/` in your system `PATH` before
  > running waybar.

```bash
go install github.com/Nadim147c/waybar-lyric@latest
```

- Or install from source

```bash
git clone https://github.com/Nadim147c/waybar-lyric.git
cd waybar-lyric
go install
```

## Usage

```
Usage: /usr/bin/waybar-lyric [options]
Get lyrics on waybar.

Options:
  -c, --compact                   Output only text content on each line
  -f, --filter-profanity string   Filter profanity from lyrics (values: full, partial)
  -i, --init                      Display JSON snippet for waybar/config.jsonc
  -o, --log-file string           Specify file path for saving logs
  -l, --lyric-only                Display only lyrics in text output
  -m, --max-length int            Set maximum character length for lyrics text (default 150)
  -q, --quiet                     Suppress all log output
  -t, --toggle                    Toggle player state between pause and resume
  -C, --tooltip-color string      Set color for inactive lyrics lines (default "#cccccc")
  -L, --tooltip-lines int         Set maximum number of lines in waybar tooltip (default 8)
  -v, --verbose                   Enable verbose logging
  -V, --version                   Display waybar-lyric version information
```

## Configuration

### Waybar Configuration

The recommended way to configure waybar-lyric is to generate the configuration
snippet using the built-in command:

```bash
waybar-lyric --init
```

This will output the proper JSON configuration snippet that you can copy directly
into your Waybar `config.jsonc` file.

### Style Example

Add to your `style.css`:

```css
#custom-lyrics {
  color: #1db954;
  margin: 0 5px;
  padding: 0 10px;
}

#custom-lyrics.paused {
  color: #aaaaaa; /* Set custom color when paused */
}
```

## Troubleshooting

If you encounter issues:

1. Check that Spotify is running and connected
2. Run with verbose logging: `waybar-lyric -v --log-file=/tmp/waybar-lyric.log`
3. Verify DBus connectivity with: `dbus-send --print-reply --dest=org.mpris.MediaPlayer2.spotify /org/mpris/MediaPlayer2 org.freedesktop.DBus.Properties.Get string:org.mpris.MediaPlayer2.Player string:PlaybackStatus`

## License

This repository is licensed under [AGPL-3.0](./LICENSE). Thanks to
[LrcLib](https://lrclib.net/) for providing lyrics.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
