{
  buildGoApplication,
  lib,
  stdenv,
  bash,
  bashInteractive,
  __includeMan ? true,
}:
buildGoApplication {
  pname = "direnv";
  version = lib.fileContents ./version.txt;
  subPackages = [ "." ];

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions (
      [
        ./go.mod
        ./go.sum
        ./gomod2nix.toml
        ./GNUmakefile
        ./stdlib.sh
        ./version.txt
        (lib.fileset.fileFilter (file: file.hasExt "go") ./.)
      ]
      ++ lib.optional __includeMan (lib.fileset.fileFilter (file: file.hasExt "1") ./man)
    );
  };

  modules = ./gomod2nix.toml;

  # we have no bash at the moment for windows
  BASH_PATH = lib.optionalString (!stdenv.hostPlatform.isWindows) "${bash}/bin/bash";

  # replace the build phase to use the GNUMakefile instead
  buildPhase = ''
    ls -la ./vendor
    make BASH_PATH=$BASH_PATH
  '';

  installPhase = ''
    echo $GOCACHE
    make install PREFIX=$out
  '';

  checkPhase = ''
    # The completion tests require builtins omitted from minimal Bash.
    export PATH="${bashInteractive}/bin:$PATH"
    make test-go
  '';

  meta = {
    description = "A shell extension that manages your environment";
    homepage = "https://direnv.net";
    license = lib.licenses.mit;
    maintainers = [ lib.maintainers.zimbatm ];
    mainProgram = "direnv";
  };
}
