---
id: install-global-scope-writes-to-home
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 1
status: done
depends-on: install-fetches-from-official-registry
branch: feature/skill-install-system
---

# Global Skills Use the User Configuration Directory

## Description

`spekk install <agent> <skill> --global` writes to `<global-config-dir>/skills/<agent>/<skill>.md`. The global configuration directory comes from `config.GlobalConfigDir()`. The skill is available to the current user in every project through the same directory that `SkillResolver` reads.

## Success Criteria

- A nonempty `XDG_CONFIG_HOME` selects `filepath.Join(XDG_CONFIG_HOME, "spekk")` as the global configuration directory. The value is used as provided, including a relative value.
- With `XDG_CONFIG_HOME` unset or empty, the global configuration directory is `filepath.Join(home, ".config", "spekk")`, where `home` comes from `os.UserHomeDir()`.
- The same rule applies on Linux and macOS. The published Windows binary uses the same rule with native path separators. The home variable is `HOME` on Linux and macOS and `USERPROFILE` on Windows. Windows binaries are not tested.
- If `os.UserHomeDir()` fails, the command reports the configuration error, including when `XDG_CONFIG_HOME` is set.
- `spekk install builder my-skill --global` writes to `filepath.Join(globalConfigDir, "skills", "builder", "my-skill.md")`.
- Missing parent directories are created with requested mode 0755. On Unix, the process umask applies.
- `SkillResolver.ResolveSkill("builder", "my-skill")` can find the installed skill through the same global configuration directory.
- The success message includes the resolved destination path. This path is absolute when the configuration base is absolute.

**Tests:** internal/install/write_test.go (`TestPerformInstall_GlobalScopeWritesUnderConfigDirAndIsResolvable`); internal/config/config_test.go (`TestResolveGlobalConfigDir_DefaultPath`, `TestResolveGlobalConfigDir_XDGOverride`).
