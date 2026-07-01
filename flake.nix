{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  outputs =
    { self, nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      inherit (nixpkgs) lib;

      perSystem =
        f:
        lib.genAttrs systems (
          system:
          let
            pkgs = import nixpkgs { inherit system; };
          in
          f { inherit lib system pkgs; }
        );
    in
    {
      packages = perSystem (
        { pkgs, system, ... }: {
          waybar-lyric = pkgs.callPackage ./nix/package.nix { };
          default = self.packages.${system}.waybar-lyric;

          ci-test = pkgs.writeShellApplication {
            name = "test";
            runtimeInputs = [ pkgs.go ];
            text = ''
              go test -v ./...
            '';
          };

          ci-lint = pkgs.writeShellApplication {
            name = "lint";
            runtimeInputs = [
              pkgs.go
              pkgs.gotools
              pkgs.fd
              pkgs.deadnix
              pkgs.golangci-lint
            ];
            text = ''
              golangci-lint run
              deadcode -test ./...
              go vet -v ./...
              fd --type file '\.nix$' --exec-batch deadnix -f {}
            '';
          };

          ci-format = pkgs.writeShellApplication {
            name = "format";
            runtimeInputs = [
              pkgs.gofumpt
              pkgs.fd
              pkgs.nixfmt
            ];
            text = ''
              gofumpt -d -e .
              fd --type file '\.nix$' --exec-batch nixfmt -c {}
            '';
          };

          ci-go-mod-tidy = pkgs.writeShellApplication {
            name = "go-mod-tidy";
            runtimeInputs = [
              pkgs.go
              pkgs.git
            ];
            text = ''
              go mod tidy
              env PAGER= git diff --exit-code
            '';
          };
        }
      );
      devShells = perSystem (
        { pkgs, ... }: {
          default = pkgs.callPackage ./nix/shell.nix { };
        }
      );
    };
}
