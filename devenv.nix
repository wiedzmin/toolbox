{ pkgs, lib, config, inputs, ... }:

{
  env = {
    PROJECTNAME = "toolbox";
    GREEN = "\\033[0;32m";
    NC = "\\033[0m"; # No Color
    GOPATH = lib.mkForce "/home/alex3rd/workspace/go"; # FIXME: try unhardcoding
  };

  scripts.hello.exec = ''echo -e "''${GREEN}welcome to $PROJECTNAME''${NC}"'';

  imports = [ inputs.nur.nixosModules.nur ];

  packages = with pkgs; with config.nur.repos; with inputs.nixpkgs-future.legacyPackages."x86_64-linux"; [
    cloc
    # gitFull
    # gitAndTools.git-crypt
    just
    tagref
    # vim
    # wiedzmin.pystdlib
  ];

  enterShell = ''
    hello
  '';

  difftastic.enable = true;

  languages.go = {
    enable = true;
    package = pkgs.go_1_23;
  };

  pre-commit.hooks = {
    golangci-lint.enable = true;
    gofmt.enable = true;
    govet.enable = true;
    typos.enable = true;
  };
}
