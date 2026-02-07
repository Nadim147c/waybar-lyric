# waybar-lyric

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/Nadim147c/waybar-lyric?style=for-the-badge&logo=go&labelColor=11140F&color=BBE9AA)](https://pkg.go.dev/github.com/Nadim147c/waybar-lyric)
[![GitHub Repo stars](https://img.shields.io/github/stars/Nadim147c/waybar-lyric?style=for-the-badge&logo=github&labelColor=11140F&color=BBE9AA)](https://github.com/Nadim147c/waybar-lyric)
[![GitHub License](https://img.shields.io/github/license/Nadim147c/waybar-lyric?style=for-the-badge&logo=gplv3&labelColor=11140F&color=BBE9AA)](./LICENSE)
[![GitHub Tag](https://img.shields.io/github/v/tag/Nadim147c/waybar-lyric?include_prereleases&sort=semver&style=for-the-badge&logo=git&labelColor=11140F&color=BBE9AA)](https://github.com/Nadim147c/waybar-lyric/tags)
[![Git Commit](https://img.shields.io/github/last-commit/Nadim147c/waybar-lyric?style=for-the-badge&logo=git&labelColor=11140F&color=BBE9AA)](https://github.com/Nadim147c/waybar-lyric/tags)

> [!IMPORTANT]
> 🔥 Found this useful? A quick star goes a long way.

A CLI tool that displays lyrics on your
[Waybar](https://github.com/Alexays/Waybar) for Linux systems.

> [!NOTE]
> If you're using waybar for the first time checkout the [example](./example/waybar/) configuration.

![Example](./example/waybar/screenshot.png)

## Description

`waybar-lyric` fetches and displays real-time lyrics on your Waybar. It provides a
scrolling lyrics display that syncs with your currently playing music, enhancing your
desktop music experience.

## Supported Players

- [Spotify](https://spotify.com)
- YouTubeMusic ([Pear-Desktop](https://github.com/pear-devs/pear-desktop))
- [Amarok](https://amarok.kde.org/)
- [Amberol](https://apps.gnome.org/en/Amberol/)
- Firefox (Specific domains)
  - `open.spotify.com`
  - `music.youtube.com`

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

- Any of the supported browser
- DBus connectivity
- [waybar](https://github.com/Alexays/Waybar)
- [go](https://go.dev/)

### Install

#### AUR

- Latest stable version

```bash
yay -S waybar-lyric
```

- The latest git commit:

```bash
yay -S waybar-lyric-git
```

#### Nixpkgs

- NixOS:

```nix
environment.systemPackages = [
  pkgs.waybar-lyric
];
```

- Home-Manager:

```nix
home.packages = [
  pkgs.waybar-lyric
];
```

- On Non NixOS:

```bash
# without flakes:
nix-env -iA nixpkgs.waybar-lyric
```

#### Manual

> You need GNU `make` and `install`

1. Build the waybar-lyric

```bash
git clone https://github.com/Nadim147c/waybar-lyric.git
cd waybar-lyric
make
```

2. Local install

```bash
make install PREFIX=$HOME/.local
```

3. Global install

```bash
sudo make install PREFIX=/usr
```

#### go install (Not recommended)

> Note: You have to make sure that `$GOPATH/bin/` in your system `PATH` before
> running waybar.

```bash
go install github.com/Nadim147c/waybar-lyric@latest
```

## Configuration

### Waybar Configuration

The recommended way to configure waybar-lyric is to generate the configuration
snippet using the built-in command:

```bash
waybar-lyric init
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

1. Check that any of the supported browser is running is running and connected
2. Run with verbose logging

```bash
waybar-lyric -v --log-file=/tmp/waybar-lyric.log
```

3. Verify DBus connectivity with:

```bash
dbus-send --print-reply \
    --dest=org.mpris.MediaPlayer2.spotify \
    /org/mpris/MediaPlayer2 org.freedesktop.DBus.Properties.Get \
    string:org.mpris.MediaPlayer2.Player \
    string:PlaybackStatus
```

## Hacking

Contributions are welcome! Feel free to submit a Pull Request.

## License

This repository is licensed under [AGPL-3.0](./LICENSE). Thanks to
[LrcLib](https://lrclib.net/) for providing lyrics.
