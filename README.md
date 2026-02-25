# claude-code-hooks

Claude Code での会話やツール操作を Obsidian ノートに自動記録するための仕組み。以下のスクリプトで構成される。

- **`claude-obsidian-save`** — Claude Code のフックから呼び出され、イベントをノートに追記する
- **`claude-obsidian-backfill-links`** — 既存ノートに session リンクをバックフィルする
- **`claude-setup-hooks`** — フック設定を `~/.claude/settings.json` に適用する
- **`claude-notify`** — macOS 通知を表示するヘルパー（WezTerm のフォーカス検出対応）

## インストール

```bash
brew tap delphinus/claude-code-hooks
brew install claude-code-hooks
claude-setup-hooks
```

開発版（HEAD）をインストールする場合:

```bash
brew install --HEAD claude-code-hooks
```

## 使い方

### claude-setup-hooks

hooks.json の内容を `~/.claude/settings.json` の `hooks` キーにマージする。`env` や `model` など端末固有の設定はそのまま保持される。

```bash
# 差分を確認（適用しない）
claude-setup-hooks --diff

# hooks を適用
claude-setup-hooks
```

### claude-obsidian-save

Claude Code の各フックイベントに応じて、セッションごとに1つの Obsidian ノート（`.md`）を作成し、時系列でイベントを追記していく。

#### 対応イベントと記録内容

| フックイベント     | 記録内容                                   |
| ------------------ | ------------------------------------------ |
| `UserPromptSubmit` | ユーザーの入力テキスト（`## User` 見出し） |
| `Stop`             | アシスタントの応答（`## Assistant` 見出し）  |
| `PostToolUse`      | ツール使用（下記参照）                     |
| `SessionEnd`       | Haiku で要約を生成しフロントマターの後に挿入 |

#### PostToolUse で記録するツール

| ツール名        | 記録形式                                            | 備考                                             |
| --------------- | --------------------------------------------------- | ------------------------------------------------ |
| `Bash`          | `> [!terminal]- description (HH:MM:SS)` + コード   | ブロックリストに該当するコマンドはスキップ        |
| `EnterPlanMode` | `> [!plan] Entering Plan Mode (HH:MM:SS)`           | related リンクの追加も試みる                       |
| `ExitPlanMode`  | `> [!plan]- タイトル (HH:MM:SS)` + プラン全文       | プランの先頭行をタイトルに使用                   |
| `Edit`          | `> [!file] Edit: path (HH:MM:SS)`                   | ファイルパスのみ（内容は記録しない）             |
| `Write`         | `> [!file] Write: path (HH:MM:SS)`                  | 同上                                             |

#### Bash コマンドのフィルタリング

ブロックリスト方式。以下のコマンドは探索用としてスキップする:

`ls`, `cat`, `head`, `tail`, `wc`, `file`, `stat`, `which`, `where`, `type`, `echo`, `printf`, `pwd`, `cd`, `test`, `true`, `false`, `grep`, `rg`, `find`, `diff`, `sort`, `uniq`, `tr`, `cut`, `mkdir`, `rmdir`, `rm`, `cp`, `mv`, `ln`, `chmod`, `chown`, `touch`, `basename`, `dirname`, `realpath`, `readlink`, `tree`, `du`, `df`, `less`, `more`, `xargs`, `tee`, `whoami`, `hostname`, `date`, `uname`, `env`, `set`, `export`, `alias`, `id`, `jq`

パイプやチェインで繋がったコマンドは、1つでも記録対象があれば全体を記録する。

#### 再開セッションのリンク

`claude -r` でセッションを再開した場合、同じ `session_id` の既存ノートを自動検出し、新しいノートにリンクを追加する。

- **frontmatter** に `related` フィールドとして `[[ファイル名]]` のリストを追加
- **本文** に `> [!link] Previous Sessions` コールアウトとして wiki-link を追加

#### ノートのファイル名規則

```
YYYYMMDD-HHMMSS-SSID-タイトル.md
```

- `SSID`: セッション ID の先頭4文字
- タイトル: 最初のユーザープロンプトの先頭50文字から生成

### claude-obsidian-backfill-links

既存の Obsidian ノートに対して、同じ `session_id` を持つノート間の `related` リンクをバックフィルする。何度実行しても安全（冪等）。

```bash
# 変更内容のプレビュー（ファイルは変更しない）
claude-obsidian-backfill-links --dry-run

# 実行
claude-obsidian-backfill-links
```

### claude-notify

macOS の通知を表示する。WezTerm 使用時は、現在のペインがフォーカスされている場合は通知を抑制する。

```bash
claude-notify 'タイトル' 'メッセージ'
```

## 前提条件

- macOS + iCloud 同期の Obsidian vault（`~/Library/Mobile Documents/iCloud~md~obsidian/Documents/Notes/Claude Code/`）
- `jq`（Homebrew で自動インストール）
- `claude` CLI（SessionEnd での要約生成に使用）

## リリース

バージョンは [Semantic Versioning](https://semver.org/) に従う。新しいバージョンをリリースするには:

```bash
git tag v1.x.x
git push origin v1.x.x
```

タグをプッシュすると GitHub Actions が自動的に以下を実行する:

1. GitHub Release を作成（リリースノートは自動生成）
2. リリースの tarball から sha256 を計算
3. `Formula/claude-code-hooks.rb` を新しいバージョンの URL と sha256 で更新
4. Formula の変更を `main` ブランチにコミット・プッシュ

リリース後は `brew upgrade claude-code-hooks` で更新できる。

## ファイル構成

```
~/.claude/
└── settings.json           # Claude Code の設定（端末固有、git 管理外）

~/.cache/claude-obsidian/
└── <session-id>            # セッションとノートの対応キャッシュ
```
