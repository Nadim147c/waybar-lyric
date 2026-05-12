{
  pkgs,
  mkGoEnv,
  gomod2nix,
  ...
}:
let
  goEnv = mkGoEnv { pwd = ../.; };
in
pkgs.mkShell {
  name = "waybar-lyric";
  buildInputs = with pkgs; [
    goEnv
    gomod2nix
    nix-fast-build
    gnumake
    gofumpt
    gopls
    golangci-lint
    waybar
  ];
}
