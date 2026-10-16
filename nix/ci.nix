{
  perSystem = { pkgs, ... }: {
    packages = {
      ci-go-mod-tidy = pkgs.writeShellApplication {
        name = "go-mod-tidy";
        runtimeInputs = with pkgs; [
          go
          git
        ];
        text = ''
          go mod tidy
          env PAGER= git diff --exit-code
        '';
      };
      ci-test = pkgs.writeShellApplication {
        name = "test";
        runtimeInputs = with pkgs; [ go ];
        text = ''
          go test -v ./...
        '';
      };
      ci-fmt = pkgs.writeShellApplication {
        name = "go-mod-tidy";
        runtimeInputs = with pkgs; [
          git
        ];
        text = ''
          nix fmt
          env PAGER= git diff --exit-code
        '';
      };
    };
  };
}
