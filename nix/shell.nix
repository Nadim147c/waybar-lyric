{pkgs ? import <nixpkgs> {}}:
pkgs.mkShell {
  name = "waybar-lyric";
  # Get dependencies from the main package
  inputsFrom = [(pkgs.callPackage ./package.nix {})];
  # Additional tooling
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
