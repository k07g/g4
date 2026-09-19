# セキュリティポリシー

## サポート対象バージョン

このプロジェクトはまだリリースタグを切っておらず、`main` ブランチの最新版のみを
サポート対象としています。

## 脆弱性の報告

セキュリティ上の問題を発見した場合は、**公開のIssueやPull Requestは作成せず**、
GitHubの [Private vulnerability reporting](https://github.com/k07g/g4/security/advisories/new)
から報告してください。

報告には以下を含めていただけると調査がスムーズです。

- 問題の概要と想定される影響
- 再現手順(可能であればリクエスト例やコードスニペット)
- 関連するファイル・エンドポイント(例: `internal/auth`, `internal/api`, `terraform/`)

対応状況や修正版の連絡は、報告いただいたSecurity Advisory上で行います。
本プロジェクトは個人開発のため、対応時間について正式なSLAは設けていませんが、
可能な限り速やかに確認・対応します。

## 対象範囲

- Goで実装されたAPIサーバー(`cmd/`, `internal/`)
- Amazon Cognito連携(`internal/auth`)
- Terraformで構築するAWSインフラ(`terraform/`)

依存パッケージ自体の既知の脆弱性はDependabotが検知します
([.github/dependabot.yml](.github/dependabot.yml))。それ以外の設計・実装上の
脆弱性(認可不備、インジェクション、シークレットの取り扱いなど)の報告を歓迎します。
