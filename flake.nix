{
  description = "Golang development environment";

  inputs = {
    flake-utils.url = github:numtide/flake-utils;
    nur.url = "/home/alex3rd/workspace/repos/github.com/wiedzmin/NUR";
    unstable.url = github:NixOS/nixpkgs/b59c06dc92f8d03660eb4155754d93a6c34cda83;
  };

  outputs = { self, flake-utils, unstable, nur }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = (import unstable {
          inherit system;
          overlays = [ nur.overlay ];
        });
      in { devShell = import ./shell.nix { inherit pkgs; }; });
}

