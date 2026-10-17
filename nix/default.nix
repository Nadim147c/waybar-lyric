{
  imports = [
    ./shell.nix
    ./treefmt.nix
    ./ci.nix
  ];
  perSystem = { pkgs, self', ... }: {
    packages = {
      waybar-lyric = pkgs.callPackage ./package.nix { };
      default = self'.packages.waybar-lyric;
    };
  };
}
