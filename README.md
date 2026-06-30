# claude-code-hooks

Claude Code での会話やツール操作を Obsidian ノートに自動記録するための Go バイナリ。6つのサブコマンドで構成される。

- **`claude-code-hooks save`** — Claude Code のフックから呼び出され、イベントをノートに追記する
- **`claude-code-hooks open`** — セッションのノートを Obsidian で開く（引数なしで現在のセッション）
- **`claude-code-hooks backfill`** — 既存ノートに session リンクをバックフィルする
- **`claude-code-hooks setup`** — フック設定を `~/.claude/settings.json` に適用する
- **`claude-code-hooks notify`** — macOS 通知を表示するヘルパー（WezTerm のフォーカス検出対応）
- **`claude-code-hooks tabcolor`** — Claude Code の状態に応じて WezTerm のタブ色を変える
- **`claude-code-hooks completion`** — シェル補完スクリプトを出力する（Bash / Zsh / Fish 対応）

## インストール

```bash
brew tap delphinus/claude-code-hooks
brew install claude-code-hooks
claude-code-hooks setup
```

開発版（HEAD）をインストールする場合:

```bash
brew install --HEAD claude-code-hooks
```

## 使い方

### claude-code-hooks setup

hooks.json の内容を `~/.claude/settings.json` の `hooks` キーにマージする。`env` や `model` など端末固有の設定はそのまま保持される。

```bash
# 差分を確認（適用しない）
claude-code-hooks setup --diff

# hooks を適用
claude-code-hooks setup
```

### claude-code-hooks save

Claude Code の各フックイベントに応じて、セッションごとに1つの Obsidian ノート（`.md`）を作成し、時系列でイベントを追記していく。

#### 対応イベントと記録内容

| フックイベント     | 記録内容                                   |
| ------------------ | ------------------------------------------ |
| `UserPromptSubmit` | ユーザーの入力テキスト（`## User` 見出し） |
| `Stop`             | アシスタントの応答（`## Assistant` 見出し）  |
| `PostToolUse`      | ツール使用（下記参照）                     |
| `SessionEnd`       | Haiku で要約を生成しフロントマターの後に挿入。終了時刻を記録（下記） |

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

#### セッションの開始・終了時刻

各ノートには開始時刻に加えて終了時刻が記録される。勤怠・稼働時間の集計に使える。

- **開始** = ファイル名 / frontmatter `date`（最初のユーザープロンプト = ノート作成時刻）。
- **終了** = frontmatter `ended`。`SessionEnd` 時に、本文中の**最後の `## Assistant (HH:MM:SS)` 見出し**（= 最終アクティビティ）を `date` の日付と合わせて記録する。セッションを開いたまま放置して閉じた場合のアイドル時間を含めないため、実作業の終わりに一致する。最終応答が翌未明にまたがる場合は日付を繰り上げる。`## Assistant` 見出しが無い場合のみ実時刻（`time.Now()`）を使う。
- 要約コールアウト `> [!summary]` の先頭にも、人間可読の `⏱ HH:MM–HH:MM (Xh Ym)`（日跨ぎは `(+1d)`）を 1 行表示する。

```yaml
date: 2026-06-29T11:28:00
ended: 2026-06-29T18:02:33
```

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

### claude-code-hooks open

セッションのノートを Obsidian で開く。セッション開始時の自動オープンは既定で無効 (`CLAUDE_OBSIDIAN_AUTO_OPEN` 参照) なので、見たいときにこのコマンドで開く。Advanced URI プラグインがあれば新しいタブで開く。

```bash
# 現在のセッションのノートを開く（実行した cwd に一致する最新ノート → 同一リポジトリ → 全体の最新）
claude-code-hooks open

# 最近のセッションノートを JSON で列挙（既定 20 件）
claude-code-hooks open --list 20

# セッション ID（完全 / 先頭一致）で開く
claude-code-hooks open <session-id>

# ノートのパスを直接指定して開く
claude-code-hooks open path/to/note.md
```

Claude Code から対話的に開きたい場合は `open-session-note` スキルを使う。

### claude-code-hooks backfill

既存の Obsidian ノートに対して、同じ `session_id` を持つノート間の `related` リンクをバックフィルする。何度実行しても安全（冪等）。

```bash
# 変更内容のプレビュー（ファイルは変更しない）
claude-code-hooks backfill --dry-run

# 実行
claude-code-hooks backfill
```

### claude-code-hooks notify

macOS の通知を表示する。WezTerm 使用時は、現在のペインがフォーカスされている場合は通知を抑制する。

```bash
claude-code-hooks notify 'タイトル' 'メッセージ'
```

### claude-code-hooks tabcolor

Claude Code の状態を WezTerm のタブ色で可視化する。各フックイベントで現在のペイン (`$WEZTERM_PANE`) の user var `claude_state` をセットし、WezTerm 側の `format-tab-title` でその値に応じてタブ背景を塗り分ける。

```bash
claude-code-hooks tabcolor <startup|thinking|idle|waiting|default>
```

- WezTerm 外（`WEZTERM_PANE` 未設定）では何もしない。
- 装飾目的なので失敗は握り潰し、フックの流れを止めない。

#### 仕組み

WezTerm には user var をセットする CLI が無いため、`claude_state` は OSC 1337 `SetUserVar` エスケープシーケンスで設定する。フックの標準出力は Claude Code にキャプチャされ `/dev/tty` も使えないことがあるので、対話実行で標準出力が端末のときはそこへ、そうでないときは `wezterm cli list` で `$WEZTERM_PANE` の tty デバイスを引いてそこへ書き込む。user var は mux 経由で GUI クライアントへ同期される。

mux 多重化環境では GUI 側のペイン ID が `$WEZTERM_PANE` と一致せず、また Claude のペインがタブのアクティブペインとは限らない。そのため `format-tab-title` 側は `active_pane` だけでなく**タブ内の全ペイン**を走査し、`claude_state` が立っているペインがあればそのタブを塗る。

#### 状態と対応イベント

| 状態 (`claude_state`) | 意味   | フックイベント                                 |
| --------------------- | ------ | ---------------------------------------------- |
| `startup`             | 起動時 | `SessionStart`                                 |
| `thinking`            | 思考中 | `UserPromptSubmit` / `PostToolUse`             |
| `idle`                | 待機中 | `Stop` / `Notification(idle_prompt)`           |
| `waiting`             | 入力待ち | `PermissionRequest` / `Notification(permission_prompt)` |
| `default`             | 非起動時 | `SessionEnd`（タブ色を通常に戻す）           |

#### WezTerm 側の設定

`wezterm.lua`（または `format-tab-title` を定義しているファイル）で user var を読んでタブを塗る。`use_fancy_tab_bar = true` でもタブ背景に反映される。`active_pane` だけでなくタブ内の全ペイン (`tab.panes`) を走査するのがポイント。

```lua
local STATE_BG = {
  startup = "#7dcfff",
  thinking = "#bb9af7",
  idle = "#9ece6a",
  waiting = "#e0af68",
}

-- タブ内のいずれかのペインに claude_state が立っていれば、その色を返す
local function claude_bg(tab)
  for _, p in ipairs(tab.panes) do
    local uv = p.user_vars
    local s = uv and uv.claude_state
    if s and STATE_BG[s] then
      return STATE_BG[s]
    end
  end
  return nil
end

wezterm.on("format-tab-title", function(tab, tabs, panes, config, hover, max_width)
  local bg = claude_bg(tab)
  local title = " " .. (tab.active_pane.title or "") .. " "
  if bg then
    return {
      { Background = { Color = bg } },
      { Foreground = { Color = "#1a1b26" } },
      { Text = title },
    }
  end
  return title
end)
```

### claude-code-hooks completion

シェル補完スクリプトを出力する。Bash、Zsh、Fish に対応。

Homebrew でインストールした場合は補完が自動的にインストールされるため、手動設定は不要。

手動で設定する場合:

```bash
# Bash（~/.bashrc に追記）
eval "$(claude-code-hooks completion bash)"

# Zsh（~/.zshrc に追記）
eval "$(claude-code-hooks completion zsh)"

# Fish（~/.config/fish/config.fish に追記）
claude-code-hooks completion fish | source
```

## 環境変数

| 変数名 | 説明 |
|---|---|
| `CLAUDE_OBSIDIAN_VAULT` | Obsidian vault パスの上書き（デフォルト: iCloud 上の `Notes/Claude Code`） |
| `CLAUDE_OBSIDIAN_AUTO_OPEN` | セッション開始時にノートを Obsidian で自動で開く（既定は開かない。値が空でなければ有効化） |

## 前提条件

- macOS + iCloud 同期の Obsidian vault（`~/Library/Mobile Documents/iCloud~md~obsidian/Documents/Notes/Claude Code/`）
- `claude` CLI（SessionEnd での要約生成に使用）

## リリース

バージョンは [Semantic Versioning](https://semver.org/) に従う。新しいバージョンをリリースするには:

```bash
git tag v2.x.x
git push origin v2.x.x
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
