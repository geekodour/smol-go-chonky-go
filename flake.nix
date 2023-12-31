{
  description = "go-postgres app";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devshell = {
      # Env vars
      # DEVSHELL_DIR: contains all the programs.
      # PRJ_ROOT: points to the project root.
      # PRJ_DATA_DIR: points to $PRJ_ROOT/.data by default.
      # NIXPKGS_PATH: path to nixpkgs source.
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
          env = [{
            name = "GOROOT";
            value = pkgs.go + "/share/go";
          }];

          # NOTE: DO NOT REMOVE
          # devshell.startup = {
          #   dummy_debug = { text = "echo ${builtins.getEnv "PRJ_DATA_DIR"}"; };
          # };

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

              # postgres
              pkgs.postgresql_16
            ]];
        };

        config.process-compose = {
          dev = {
            settings.processes = {
              postgres-server = {
                command = ''
                  pg_ctl -o "-p 7777 -k $PRJ_DATA_DIR/pg_sock" -D $PRJ_DATA_DIR/pg start'';
                is_daemon = true;
                shutdown = {
                  command = ''
                    pg_ctl -o "-p 7777 -k $PRJ_DATA_DIR/pg_sock" -D $PRJ_DATA_DIR/pg stop'';
                };
              };
            };
          };
        };

      };

      # Usual flake attributes if any
      flake = { };
    };
}
