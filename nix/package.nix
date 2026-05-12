# Do not call this package with regular arguments
{
  src,
  lib,
  stdenv,
  installShellFiles,
  buildGoApplication,
}:
buildGoApplication rec {
  pname = "waybar-lyric";
  version = "0-unstable";

  inherit src;
  modules = ../gomod2nix.toml;

  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
  ];

  nativeBuildInputs = [ installShellFiles ];
  postInstall = lib.optionalString (stdenv.buildPlatform.canExecute stdenv.hostPlatform) /* bash */ ''
    installShellCompletion --cmd waybar-lyric \
      --bash <($out/bin/waybar-lyric _carapace bash) \
      --fish <($out/bin/waybar-lyric _carapace fish) \
      --zsh <($out/bin/waybar-lyric _carapace zsh)
  '';

  meta = {
    description = "Waybar module for displaying song lyrics";
    homepage = "https://github.com/Nadim147c/waybar-lyric";
    license = lib.licenses.agpl3Only;
    mainProgram = "waybar-lyric";
    platforms = lib.platforms.linux;
  };
}
