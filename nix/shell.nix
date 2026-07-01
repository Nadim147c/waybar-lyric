{ pkgs }:
pkgs.mkShell {
  name = "waybar-lyric";
  buildInputs = [
    pkgs.nix-fast-build
    pkgs.gnumake
    pkgs.gofumpt
    pkgs.gopls
    pkgs.golangci-lint
    pkgs.waybar
  ];
}
