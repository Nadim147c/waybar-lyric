{ pkgs, ... }:
pkgs.mkShell {
  name = "waybar-lyric";
  inputsFrom = [ (pkgs.callPackage ./package.nix { }) ];
  buildInputs = with pkgs; [
    gnumake
    go
    gofumpt
    gopls
    golangci-lint
    waybar
  ];
}
