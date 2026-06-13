# Agent Instructions / Project Config Role Plan

この文書は、XELYON の `AGENTS.md` / `CLAUDE.md` / `xelyon.yaml` / global guidance の役割を整理するための内部計画書である。
公開 docs ではなく、実装前の壁打ち・設計・Goal handoff の source of truth として使う。

## 0. Purpose

XELYON の project guidance と project config の導線を、業界標準に近い `AGENTS.md` first に寄せる。

目的は次の 4 点。

- 初心者には `AGENTS.md` を主導線として見せ、どこに作業方針を書くべきか迷わせない。
- 既存 Claude / Codex ユーザーは `CLAUDE.md` や `~/.codex/AGENTS.md` を明示選択で再利用できるようにする。
- `xelyon.yaml` は XELYON 固有の構造化 project config に寄せ、一般的な長文 instruction の主導線から外す。
- 「何も読まない」「1つだけ読む」「複数読む」を config / UI で明確に選べるようにする。

## 1. Current Source Findings

source-confirmed facts only.

- `internal/config/config_types.go` の `AgentInstructionsConfig` は `project`, `global`, `include_local_files`, `expand_imports`, `max_file_bytes`, `max_total_bytes` を持つ。
- `internal/config/defaults_sections.go` の project guidance 候補は `AGENTS.md`, `CLAUDE.md`, `.claude/CLAUDE.md`。
- 現在の作業差分では global guidance default を `enabled: true`、候補を `~/.xelyon/AGENTS.md` のみに変更した。
- 現在の作業差分では `SaveConfig` / 初回 config 作成時に `~/.xelyon/AGENTS.md` を空ファイルで作る。既存ファイルは上書きしない。
- `internal/config/project_instructions.go` の `project.mode=fallback` は `xelyon.yaml` がある場合に project guidance を読まない。
- `project.mode=always` は `xelyon.yaml` がある場合でも project guidance を advisory として読む。
- `docs/config.md` と `config.yaml.example` は generator 管理で、config default を変えたら `make gen-all` が必要。
- 現行 README / usage docs は `xelyon.yaml` を project context / rules の主導線として説明している箇所がある。

## 2. Product Direction

### 2.1 Primary Guidance

`AGENTS.md` を project guidance の primary にする。

新規ユーザーへの説明は原則として次の形にする。

```text
作業方針や開発ルールは AGENTS.md に書く。
XELYON 固有の構造化設定は xelyon.yaml に書く。
```

`xelyon.yaml` からは、現在の `context` / `rules` を主導線として案内する役割を外す。
既存 file の読み込み互換は残すが、新規 template / docs / `/init` では `context` / `rules` を推奨しない。

### 2.2 Compatibility Guidance

`CLAUDE.md` と `.claude/CLAUDE.md` は compatibility として読み込み候補に残す。
ただし、新規作成の推奨先にはしない。
project guidance の default files は `AGENTS.md` のみにする。
`CLAUDE.md` / `.claude/CLAUDE.md` は `/config` で明示選択できる互換候補として扱う。

`~/.codex/AGENTS.md` や `~/.claude/CLAUDE.md` は既存資産の再利用口として選べるようにする。
ただし default で勝手に読む対象にはしない。

### 2.3 Global Guidance

XELYON の default global guidance は `~/.xelyon/AGENTS.md` にする。

default は次の意味にする。

```yaml
agent_instructions:
  global:
    enabled: true
    files:
      - ~/.xelyon/AGENTS.md
```

初回は空ファイルでよい。
存在しない場合は XELYON が空で用意する。
存在する場合は絶対に上書きしない。

### 2.4 xelyon.yaml

`xelyon.yaml` は repo-local XELYON config に寄せる。

役割:

```text
xelyon.yaml = repo 固有の XELYON 設定
AGENTS.md = repo 固有の agent instructions
~/.xelyon/config.yaml = user global XELYON 設定
~/.xelyon/AGENTS.md = user global agent instructions
```

主な用途:

- `ignore`: Project Map / `list_dir` / `search_code` で共有する ignore pattern
- `final_checks`: project 固有の final checks / future hooks override
- `conditional`: path 条件付きで XELYON が出し分ける必要がある短い rules/context
- XELYON 固有 runtime override: provider history reduction など

`xelyon.yaml` に長い一般 instruction、README 的な説明、ディレクトリ構造、関数目次を書かせない。
`context` / `rules` は legacy compatibility として読み続けるが、new template からは外す。
new template からは `AI 用コンテキスト。ドキュメントではありません。` や `AI が許可なくこのファイルを肥大化させることを禁止します。` のような instruction 的コメントも外す。
`xelyon.yaml` の header/comment は repo-local XELYON config であることだけを短く示す。
将来 `hooks` を追加する場合、global hooks は `~/.xelyon/config.yaml`、repo override は `xelyon.yaml` に置く。
hooks 設計は今回の導線整理に混ぜず、別タスクに切る。

## 3. Non-goals

- `xelyon.yaml` をすぐ削除しない。
- 既存 `xelyon.yaml` の `context` / `rules` をいきなり破壊的に無効化しない。
- `~/.codex/AGENTS.md` や `~/.claude/CLAUDE.md` を default で勝手に読む対象にしない。
- `CLAUDE.md` 互換を消さない。
- `AGENTS.md` を YAML schema 化しない。普通の Markdown として扱う。
- guidance 読み込みのためだけに unsafe な path escape や untracked file 読み込み default を増やさない。

## 4. Responsibility Boundaries

```text
guidance file format:
  owner: AGENTS.md / CLAUDE.md Markdown
  role: human-readable instruction

agent_instructions config:
  owner: ~/.xelyon/config.yaml
  source files: internal/config/config_types.go, defaults_sections.go, validator_agent_instructions.go
  role: which guidance files to read, how much to read, import/local-file policy

project config:
  owner: xelyon.yaml
  source files: internal/config/project*.go
  role: repo-local XELYON config, not general agent instructions

guidance loader:
  source files: internal/config/project_instructions*.go
  role: resolve root, enforce path boundary, git tracking policy, byte budget, import expansion

config docs/generated:
  source files: scripts/config_sections.go, scripts/gen-config-*.go
  generated outputs: config.yaml.example, docs/config.md, internal/config/registry_generated.go

TUI / command UX:
  source files: internal/agent/init.go, internal/tui/*project*, internal/tui/*config*
  role: first-run guidance creation, /project, /config editing surface
```

## 5. Target Config Contract

`mode` / `enabled` は「読むかどうか」、`files` は「何を読むか」を表す。

```yaml
agent_instructions:
  project:
    mode: always
    files:
      - AGENTS.md

  global:
    enabled: true
    files:
      - ~/.xelyon/AGENTS.md
```

`project.mode` の product default は `always` にする。
`fallback` は legacy compatibility mode として読み込み互換を残すが、主導線や `/config` の通常 UI では前面に出さない。

選択例:

```yaml
# project guidance を読まない
project:
  mode: off
  files:
    - AGENTS.md
```

```yaml
# project AGENTS.md だけ読む
project:
  mode: always
  files:
    - AGENTS.md
```

```yaml
# Claude 互換も読む
project:
  mode: always
  files:
    - AGENTS.md
    - CLAUDE.md
    - .claude/CLAUDE.md
```

```yaml
# 既存 Codex global を再利用する
global:
  enabled: true
  files:
    - ~/.codex/AGENTS.md
```

```yaml
# global guidance を読まない
global:
  enabled: false
  files:
    - ~/.xelyon/AGENTS.md
```

## 6. UX Direction

### First-run

新規ユーザーには `AGENTS.md` を project guidance の主導線として案内する。
`~/.xelyon/AGENTS.md` は個人 global guidance の置き場として空で用意する。

### /init

`/init` の主目的は project `AGENTS.md` creation に変える。

target behavior:

```text
AGENTS.md がない:
  -> AGENTS.md を作る
  -> "Created AGENTS.md" と出す

AGENTS.md がある:
  -> 何も書き換えない
  -> "AGENTS.md already exists. Left unchanged." と出す
  -> 次にできる操作を短く出す
```

`CLAUDE.md` だけがある場合も、`CLAUDE.md` を勝手にコピーしない。
`AGENTS.md` を新規作成し、`CLAUDE.md` は compatibility guidance として `/config` で選べることを案内する。

`AGENTS.md` は人間が編集する instruction file なので、template 再生成や `--force` overwrite を初期実装に入れない。

target command split:

```text
/init
  -> AGENTS.md を作る

/project
  -> xelyon.yaml を作る / 編集する
```

互換のため、既存の `xelyon.yaml` template 作成導線をすぐ消す必要はない。
ただし docs 上の「最初に作るもの」は `AGENTS.md` に寄せたい。

### /config UI

guidance file selection は checkbox 的に扱う。

```text
Project guidance
Mode: Always / Fallback / Off
[x] AGENTS.md
[ ] CLAUDE.md
[ ] .claude/CLAUDE.md
[+] Add custom project path

Global guidance
[x] Enable global guidance
[x] ~/.xelyon/AGENTS.md
[ ] ~/.codex/AGENTS.md
[ ] ~/.xelyon/CLAUDE.md
[ ] ~/.claude/CLAUDE.md
[+] Add custom global path
```

`files` にないものは読まない。
存在しない file は warning ではなく skip とする。
実際に読んだ guidance file は status / context size / startup 表示で確認できるようにする。

## 6.1 Project Template Direction

new `xelyon.yaml` template は repo-local XELYON config の最小例にする。

`context` / `rules` は含めない。
AI edit ban comment も含めない。

`final_checks` は空の実設定として置かない。
現在の contract では空の project `final_checks` が global fallback を無効化する override になり得るため、template ではコメント例に留める。

recommended template shape:

```yaml
# XELYON repo config

# ignore:
#   patterns:
#     - dist
#     - generated

# final_checks:
#   commands:
#     - make ci-check
#   timeout: 600
```

## 6.2 Legacy xelyon.yaml Context / Rules

既存 `xelyon.yaml context/rules` は読み込み互換を残す。
通常起動で warning は出さない。

移行は docs と `/init` の導線で自然に促す。
長すぎ warning や auto-migration は今回やらない。

## 6.3 Loaded Guidance Observability

status / context size / startup のいずれかで、実際に読んだ guidance file を確認できるようにする。

候補:

```text
Project guidance: AGENTS.md
Global guidance: ~/.xelyon/AGENTS.md (empty)
```

存在しない候補は通常表示しない。
空ファイルは `empty` として表示する。

## 7. Implementation Priority

1. Current diff closure: global default `~/.xelyon/AGENTS.md` and empty file bootstrap.
2. Change project guidance default to `mode: always` and `files: [AGENTS.md]`.
3. Keep `CLAUDE.md` / `.claude/CLAUDE.md` as selectable compatibility candidates, not default project files.
4. Remove `context` / `rules` from new `xelyon.yaml` template and docs as recommended fields.
5. Update `/init` to create project `AGENTS.md` without overwriting existing files.
6. Update docs / README / usage to `AGENTS.md` first and `xelyon.yaml` as repo-local XELYON config.
7. Adjust `/project` command wording / behavior for xelyon.yaml.
8. Add `/config` selection UX for project/global guidance files.
9. Add observability for loaded guidance files.
10. Backward compatibility warnings or migration notes for legacy `xelyon.yaml context/rules`.
11. Leave future hooks design as a separate task.

## 8. Open Decisions

現時点で、導線整理に必要な product decisions は確定済み。

Hooks 設計、ignore 設定体系の整理、legacy `context/rules` の warning / migration は別タスクで扱う。

## 9. Tests

Focused tests:

- default config contract: global enabled true, global files only `~/.xelyon/AGENTS.md`
- default config contract: project mode `always`, project files only `AGENTS.md`
- legacy compatibility: explicit project mode `fallback` still behaves as legacy fallback
- compatibility selection: `CLAUDE.md` and `.claude/CLAUDE.md` can still be loaded when present in project files
- bootstrap: first `LoadConfig` creates empty `~/.xelyon/AGENTS.md`
- no overwrite: existing `~/.xelyon/AGENTS.md` content is preserved
- project mode: `fallback` / `always` / `off` behavior with and without `xelyon.yaml`
- global file selection: one file / multiple files / disabled / missing file skip
- custom global path: `~/.codex/AGENTS.md` can be selected and loaded when configured
- `/init`: creates `AGENTS.md` when missing, does not overwrite existing `AGENTS.md`, does not copy `CLAUDE.md`
- `xelyon.yaml` template: does not include recommended `context` / `rules`; focuses on repo-local XELYON config such as `ignore`, `final_checks`, and future hooks
- `xelyon.yaml` template: does not include instruction-style comments such as AI edit bans or AI-context warnings
- `xelyon.yaml` template: keeps `final_checks` as commented example, not empty active override
- loaded guidance observability: shows loaded project/global guidance and marks empty global AGENTS.md as empty
- docs/generated consistency: `make gen-all`

Broader verification:

```sh
go test ./internal/config ./scripts/internal/configgen
go test ./internal/agent
go test ./cmd
make ci-check
```

## 10. Goal Handoff Prompt

```text
/goal Implement docs/dev/agent-instructions-project-config-role-plan.md end to end.

Use docs/dev/agent-instructions-project-config-role-plan.md as the source of truth. Preserve the role split: AGENTS.md is the primary human/project guidance, CLAUDE.md is compatibility, ~/.xelyon/AGENTS.md is XELYON global guidance, and xelyon.yaml is XELYON structured project config. Keep existing xelyon.yaml compatibility unless the plan explicitly changes it. Re-read the plan after resume or context compaction. Do not commit or push unless explicitly requested.
```
