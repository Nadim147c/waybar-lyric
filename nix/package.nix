{
  lib,
  stdenv,
  versionCheckHook,
  installShellFiles,
  buildGoModule,
}:
let
  inherit (lib) optionalString cleanSource;
  inherit (lib.fileset) toSource unions;
in
buildGoModule rec {
  pname = "waybar-lyric";
  version = "0-unstable";

  src = cleanSource (toSource {
    root = ../.;
    fileset = unions [
      ../cmd
      ../internal
      ../ascii.go
      ../ascii.txt
      ../main.go
      ../go.mod
      ../go.sum
    ];
  });

  vendorHash = "sha256-9hXIWrqYExXHfIntXCUbrSkhjmLjM163UOHRq/odaAA=";

  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
  ];

  nativeBuildInputs = [ installShellFiles ];
  postInstall = optionalString (stdenv.buildPlatform.canExecute stdenv.hostPlatform) /* bash */ ''
    installShellCompletion --cmd waybar-lyric \
      --bash <($out/bin/waybar-lyric _carapace bash) \
      --fish <($out/bin/waybar-lyric _carapace fish) \
      --zsh <($out/bin/waybar-lyric _carapace zsh)
  '';

  doInstallCheck = true;
  nativeInstallCheckInputs = [ versionCheckHook ];
  versionCheckKeepEnvironment = [ "XDG_CACHE_HOME" ];
  preInstallCheck = ''
    # ERROR Failed to find cache directory
    export XDG_CACHE_HOME=$(mktemp -d)
  '';

  meta = {
    description = "Waybar module for displaying song lyrics";
    homepage = "https://github.com/Nadim147c/waybar-lyric";
    license = lib.licenses.agpl3Only;
    mainProgram = "waybar-lyric";
    platforms = lib.platforms.linux;
  };
}
