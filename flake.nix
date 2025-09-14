{
  description = "Obsidian MCP Server development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          name = "obsidian-mcp-server-dev";

          buildInputs = with pkgs; [
            # Go development tools
            go
            gopls
            go-tools
            gotests
            gomodifytags
            gore
            gofumpt

            # Linting and code quality
            golangci-lint

            # General development tools
            git
            curl
            jq

            # Testing tools
            delve # Go debugger

            # Build tools
            gnumake
          ];
        };
      });
}
