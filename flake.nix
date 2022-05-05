{
  description = "uncommitted-go";

  inputs = {
    nixpkgs.url = "nixpkgs/nixpkgs-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    utils.lib.eachDefaultSystem
      (system:
        with import nixpkgs { inherit system; }; rec {
          packages.default = buildGoModule rec {
            name = "uncommitted-go";
            pname = name;
            src = ./.;
            vendorSha256 = "euft4bcFO3VFbubN50IDBlYY2zzEmawDzwmpbZsUiCA=";
          };

          devShells.default = mkShell { nativeBuildInputs = [ go gopls ]; };
        }) // {
      overlays.default = (final: _: {
        uncommitted-go = self.packages."${final.system}".default;
      });
    };
}
