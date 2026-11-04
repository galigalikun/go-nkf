# go-nkf

`nkf` に似た引数で文字コード判定・変換を実行するGo製CLIです。
外部の `nkf` コマンドやCGOは不要です。
日本語の文字コード変換には `golang.org/x/text` を使用します。

## ビルドと使い方

```sh
go build -o go-nkf .

# 自動判定してUTF-8へ変換
./go-nkf -w testdata/test.shift-jis.txt > converted.txt

# 入力をShift_JISと指定し、UTF-8・LFへ変換（短い引数は連結可能）
./go-nkf -SwLu testdata/test.shift-jis.txt > converted.txt

# 入出力の文字コード名を指定
./go-nkf --ic=UTF-8 --oc=CP932 testdata/test.utf-8.txt > converted.sjis.txt

# パイプ・標準入力にも対応
cat testdata/test.euc-jp.txt | ./go-nkf -w

# 文字コードと改行形式を表示
./go-nkf --guess testdata/test.shift-jis.txt
# Shift_JIS (LF)

# 文字コード名だけを表示
./go-nkf -g testdata/test.utf-8.txt
# UTF-8

./go-nkf --help
```

ファイルを指定しなければ標準入力を読み、結果は標準出力へ書き出します。
`-` でも標準入力を指定できます。複数ファイルの変換結果は順番に連結され、
判定モードではファイル名を付けて表示されます。
`--` 以降は `-` で始まる名前もファイル名として扱います。
出力文字コードを指定しない場合は ISO-2022-JP になります。
エラーは標準エラー出力へ表示し、終了コードは非ゼロになります。

## 対応オプション

| オプション | 動作 |
| --- | --- |
| `-w` / `-w8` / `-w80` | UTF-8出力（BOMなし / あり / なし） |
| `-s` / `-e` / `-j` | Shift_JIS / EUC-JP / ISO-2022-JP出力 |
| `-W` / `-S` / `-E` / `-J` | UTF-8 / Shift_JIS / EUC-JP / ISO-2022-JP入力を指定 |
| `-w16[BL][0]` / `-w32[BL][0]` | UTF-16 / UTF-32出力。既定はBE・BOMあり、`L` はLE、`0` はBOMなし |
| `-W16[BL]` / `-W32[BL]` | UTF-16 / UTF-32入力を指定。既定はBE、BOMがあればそのバイト順を使用 |
| `--ic=名前` / `--oc=名前` | 入力 / 出力の文字コードを指定。`--ic 名前` という指定も可能 |
| `-g` / `--guess=1` | 文字コード名を表示 |
| `-g2` / `--guess` / `--guess=2` | 文字コード・BOM・改行形式を表示 |
| `-Lu` / `-Lw` / `-Lm` | LF / CRLF / CRに改行を統一 |
| `-d` / `-c` / `-L0` | LFに変換 / CRLFに変換 / 改行形式を保持 |
| `-x` / `-X` | 半角カナを保持 / 全角化（既定は全角化） |
| `-t` | 入力バイト列をそのまま出力（判定オプションとの併用は不可） |
| `--help` / `--version` / `-v` | ヘルプ / バージョン情報 |

`--jis` / `--sjis` / `--euc` と、対応する `--jis-input` /
`--sjis-input` / `--euc-input` も使用できます。
同じ設定を複数回指定すると、後の指定が優先されます。

文字コード名は大文字・小文字を区別しません。
`ASCII`、`UTF-8`、`UTF-8N`、`UTF-8-BOM`、`Shift_JIS`、`SJIS`、
`CP932`、`Windows-31J`、`EUC-JP`、`ISO-2022-JP`、
`UTF-16[BE|LE][-BOM]`、`UTF-32[BE|LE][-BOM]` に対応します。
`--oc=UTF-16` / `--oc=UTF-32` はBE・BOMなしです。

## 互換性の範囲

[nkfのオプション仕様](https://github.com/nurse/nkf/blob/master/README.md)のうち、
上記の範囲に対応しています。すべての機能を再現するものではありません。

- MIME、行折り返し、`--overwrite` / `--in-place` などの未対応オプションはエラーになります。
- 入力の自動判定はBOM、JISのエスケープシーケンス、UTF-8の妥当性、
  EUC-JP / Shift_JISの復号結果を使う推定です。曖昧なデータには `--ic` を指定してください。
  BOMなしのUTF-16 / UTF-32は入力文字コードの明示指定が必要です。
- Shift_JISには `x/text` のCP932互換の実装を使います。
  nkf固有のマッピング・拡張文字判定は再現しません。
- 不正な入力や、出力文字コードで表現できない文字はエラーにします。
  文字を黙って削除するフォールバックは行いません。
  ISO-2022-JPへの半角カナ出力（`-x -j`）もこの制限に含まれます。
- 自動判定で復号できないデータは判定モードで `BINARY` と表示します。
- 各入力をメモリに読み込んで処理します。

既存の内部関数 `guess(fileName)` / `guessenc(fileName)` は従来の
UTF-8 / ASCII検証を維持します。CLIの `--guess` は上記の自動判定を使います。

## 開発

```sh
go test ./...
go test -race ./...
go vet ./...
```
