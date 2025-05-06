{ pkgs, inputs, ... }:
{
  packages = with pkgs; [
    pre-commit
  ];

  languages.go = {
    enable = true;
  };

  enterShell = ''
    # Ensure pre-commit hook is installed/updated on direnv/devenv entry
    if [ -d .git ]; then
      pre-commit install --install-hooks --overwrite || true
    fi
  '';
}