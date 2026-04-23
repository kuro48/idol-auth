{
  description = "Nix tooling for idol-auth development and production operations";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        deployInputs = with pkgs; [
          bash
          coreutils
          curl
          docker-client
          docker-compose
          go_1_26
          gnumake
          gnused
          gnugrep
          jq
          openssl
          perl
        ];
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; deployInputs ++ [
            git
            go_1_26
            postgresql_17
            redis
          ];

          shellHook = ''
            echo "idol-auth nix shell"
            echo "  make test"
            echo "  nix run .#render-production-config"
            echo "  nix run .#deploy-production -- .env.production"
          '';
        };

        apps = {
          render-production-config = {
            type = "app";
            program = "${pkgs.writeShellApplication {
              name = "render-production-config";
              runtimeInputs = with pkgs; [ bash coreutils perl ];
              text = ''
                exec "$PWD/scripts/render-production-config.sh" "$@"
              '';
            }}/bin/render-production-config";
          };

          config-check = {
            type = "app";
            program = "${pkgs.writeShellApplication {
              name = "config-check";
              runtimeInputs = with pkgs; [ bash go_1_26 ];
              text = ''
                exec go run ./cmd/configcheck "$@"
              '';
            }}/bin/config-check";
          };

          deploy-production = {
            type = "app";
            program = "${pkgs.writeShellApplication {
              name = "deploy-production";
              runtimeInputs = deployInputs;
              text = ''
                if [[ $# -gt 0 ]]; then
                  export ENV_FILE="$1"
                  shift
                fi

                exec "$PWD/scripts/deploy-production.sh" "$@"
              '';
            }}/bin/deploy-production";
          };

          backup-postgres = {
            type = "app";
            program = "${pkgs.writeShellApplication {
              name = "backup-postgres";
              runtimeInputs = with pkgs; [
                bash
                coreutils
                docker-client
                docker-compose
                gzip
              ];
              text = ''
                if [[ $# -gt 0 ]]; then
                  export ENV_FILE="$1"
                  shift
                fi

                exec "$PWD/scripts/backup-postgres.sh" "$@"
              '';
            }}/bin/backup-postgres";
          };
        };
      });
}
