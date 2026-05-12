{

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      gomod2nix,
      ...
    }:
    (flake-utils.lib.eachSystem [ "x86_64-linux" "aarch64-linux" ] (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        inherit (pkgs) callPackage lib;
        gomod = gomod2nix.legacyPackages.${system};

        src = lib.cleanSource (
          lib.fileset.toSource {
            root = ./.;
            fileset = lib.fileset.unions [
              ./.editorconfig
              ./.golangci.yaml
              ./cmd
              ./internal
              ./ascii.go
              ./ascii.txt
              ./main.go
              ./go.mod
              ./go.sum
            ];
          }
        );

        mkGoTest = lib.mapAttrs' (
          name: value:
          lib.nameValuePair name (
            gomod.buildGoApplication {
              inherit name src;
              dontBuild = true;
              modules = ./gomod2nix.toml;
              doCheck = true;
              nativeBuildInputs = with pkgs; [
                go
                golangci-lint
                gofumpt
                writableTmpDirAsHomeHook
              ];
              checkPhase = value;
              installPhase = ''
                touch "$out"
              '';
            }

          )
        );

      in
      {
        checks = mkGoTest {
          go-test = /* bash */ ''
            go test -v ./...
          '';
          go-vet = /* bash */ ''
            go vet -v ./...
          '';
          go-lint = /* bash */ ''
            golangci-lint run
          '';
          go-format = /* bash */ ''
            gofumpt -d -e .
          '';
        };
        packages.default = callPackage ./nix/package.nix {
          inherit (gomod) buildGoApplication;
          inherit src;
        };
        devShells.default = callPackage ./nix/shell.nix {
          inherit (gomod) mkGoEnv gomod2nix;
        };
      }
    ));
}
