{ lib, ... }:
let
  inherit (lib) mkForce;
in
{
  perSystem =
    { pkgs, ... }:
    let
      # Fixes: named files must all be in one directory
      makeGoPositionalArgsFixer =
        package: bin:
        pkgs.writers.writeNu "treefmt-go-positional-arg-fixer" /* nu */ ''
          def --wrapped main [...p: string] {
            mut files = []
            mut args = []

            for arg in $p {
              if ($arg =~ '\.go$') and ($arg | path exists) {
                $files = $files ++ [$arg]
              } else {
                $args = $args ++ [$arg]
              }
            }

            let add_dot_slash = {|it| "./" + $it}

            let packages = (
              $files
              | path expand
              | path parse
              | group-by parent
              | transpose
              | rename parent child
              | each {|it|
                let files = $it.child
                  | path join
                  | path relative-to $env.PWD
                  | each $add_dot_slash
                let package = $it.parent
                  | path relative-to $env.PWD
                  | default --empty "."
                  | each $add_dot_slash
                return {files: $files package: $package}
              }
            )

            for package in $packages {
              ${bin} ...$args ${if package then "$package.package" else "...$package.files"}
            }
          }
        '';

      passGoPackages = makeGoPositionalArgsFixer true;
      passGoFiles = makeGoPositionalArgsFixer false;
    in
    {
      treefmt = {
        programs = {
          actionlint.enable = true;
          golangci-lint.enable = true;
          gofumpt.enable = true;
          deadnix.enable = true;
          dos2unix.enable = true;
          nixfmt.enable = true;
          deno.enable = true;
          jq.enable = true;
        };
        settings.formatter = {
          gofix = {
            includes = [ "*.go" ];
            options = [ "fix" ];
            stdin-options = [ ];
            command = "${passGoFiles "${pkgs.go}/bin/go"}";
          };
          govet = {
            includes = [ "*.go" ];
            options = [ "vet" ];
            stdin-options = [ ];
            command = "${passGoPackages "${pkgs.go}/bin/go"}";
          };
          golangci-lint.command = mkForce "${passGoPackages "${pkgs.golangci-lint}/bin/golangci-lint"}";
        };
      };
    };

}
