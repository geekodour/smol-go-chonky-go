{
  description = "go-postgres app";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devshell = {
      url = "github:numtide/devshell";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    process-compose-flake.url = "github:Platonic-Systems/process-compose-flake";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devshell.flakeModule
        # https://community.flake.parts/process-compose-flake
        inputs.process-compose-flake.flakeModule
      ];
      systems = [ "x86_64-linux" ];
      perSystem = { config, self', inputs', pkgs, lib, system, ... }: {

        config.devshells.default = {
          name = "yolo";

          devshell.startup = {
            pg_setup = {
              text = ''
                mkdir -p $PGSOCK
                if [ ! -d $PGDATA ]; then
                  initdb -U postgres $PGDATA --auth=trust >/dev/null
                fi
                cp $PROJECT_ROOT/.devops/pg/postgresql.conf $PGDATA/postgresql.conf
              '';
            };
          };

          motd = ''
            ┈┈┈┈▕▔╱▔▔▔━▁
            ┈┈┈▕▔╱╱╱👁┈╲▂▔▔╲
            ┈┈▕▔╱╱╱╱💧▂▂▂▂▂▂▏
            ┈▕▔╱▕▕╱╱╱┈▽▽▽▽▽
            ▕▔╱┊┈╲╲╲╲▂△△△△
            ▔╱┊┈╱▕╲▂▂▂▂▂▂╱
            ╱┊┈╱┉▕┉┋╲┈
          '';

          packages = with lib;
            mkMerge [[
              # golang
              pkgs.go
              pkgs.gotestsum
              pkgs.gofumpt
              pkgs.golangci-lint
              pkgs.sqlc

              # postgres
              pkgs.postgresql_16
              pkgs.pgcli

              # ops
              pkgs.goose
              pkgs.k6
              pkgs.prometheus
            ]];
        };

        config.process-compose = {
          dev = {
            settings.processes = {
              postgres-server = {
                command =
                  ''pg_ctl -o "-p $PGPORT -k $PGSOCK" -D $PGDATA start'';
                is_daemon = true;
                shutdown = { command = "pg_ctl -D $PGDATA stop"; };
              };
            };
          };
        };

      };

      # Usual flake attributes if any
      flake = { };
    };
}
