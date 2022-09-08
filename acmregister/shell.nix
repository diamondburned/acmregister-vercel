{ systemPkgs ? import <nixpkgs> {} }:

let lib = systemPkgs.lib;
	pkgs =
		if (lib.versionAtLeast systemPkgs.go.version "1.19")
		then systemPkgs
		else import (systemPkgs.fetchFromGitHub {
			owner = "NixOS";
			repo  = "nixpkgs";
			rev   = "e105167e98817ba9fe079c6c3c544c6ef188e276";
			hash  = "sha256:1274abx6ibdlavvm43a398rkb3fnhr1s5n7fjiv9l9zzpjgrdyvq";
		}) {};

	sqlc = pkgs.buildGoModule {
		name = "sqlc";
		version = "1.15.0";

		src = pkgs.fetchFromGitHub {
			owner  = "kyleconroy";
			repo   = "sqlc";
			rev    = "v1.15.0";
			sha256 = "11iinay0din8rjd20sbjipqvsvarw6r0sd5vgfqjq5bcx41vkxji";
		};

		proxyVendor = true;
		vendorSha256 = "0ycvjn6yfdijr2avhw16ayn0s1lcmgvxd901l0129ks2xbhlbar9";

		# Skip fuzz tests.
		doCheck = false;
	};

	sql-formatter = systemPkgs.writeShellScriptBin "sql-formatter" ''
		set -e

		[[ "$1" ]] && stdin=$(< "$1") || stdin=$(cat)
		language=$(command grep -oP '^-- Language: \K.*$' <<< "$stdin")

		config_for_lang() {
			cat<<EOF
				{
					"expressionWidth": 50,
					"keywordCase": "upper",
					"language": "$1",
					"tabulateAlias": false,
					"useTabs": true
				}
EOF
		}

		${pkgs.nodePackages.sql-formatter}/bin/sql-formatter \
			--config <(config_for_lang "$language") <<< "$stdin"
	'';

in pkgs.mkShell {
	buildInputs = with pkgs; [
		go
		gotools
		gopls
		sqlc
		sql-formatter
	];
}
