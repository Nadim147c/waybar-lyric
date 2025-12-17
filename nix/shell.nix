{ pkgs, ... }:
pkgs.mkShell {
  name = "waybar-lyric";
  inputsFrom = [ (pkgs.callPackage ./package.nix { }) ];
  buildInputs = with pkgs; [
    gnumake
    go
    gofumpt
    golines
    gopls
    revive
    waybar
  ];
}
