{
  perSystem = { pkgs, ... }: {
    devShells.default = pkgs.mkShell {
      name = "waybar-lyric";
      buildInputs = with pkgs; [
        nix-fast-build
        gnumake
        gofumpt
        gopls
        golangci-lint
        waybar
      ];
    };
  };

}
