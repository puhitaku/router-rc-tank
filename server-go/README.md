# サーバー実装 (Go) について

ここには、`server` ディレクトリにある MicroPython 実装とほぼ同じ動作をする Go 実装のサーバーがあります。


# 使用方法

1. Go をインストールする

2. `make` を実行して `server-go` バイナリが生成されるか確認する

2. 実機に転送する（ストレージに対してバイナリサイズが大きいため `/tmp` 以下に転送する）

    ```sh
    $ scp server-go root@{address_of_the_router}:/tmp/
    ```

3. 実機で実行する

    ```sh
    $ ssh root@{address_of_the_router} /tmp/server-go
    ```

4. リクエストを送る

    ```sh
    $ curl {address_of_the_router}:8080/healthz
    {"message":"I'm as ready as I'll ever be!"}

    $ curl -X PUT -H "Content-Type: application/json" -d '{"operation": "f"}' {address_of_the_router}:8080/operation
    {"operation":"f","error":null}
    ```

# 仕様

ポート番号: 8080

|パス|メソッド|Content-Type (PUT)|リクエストボディ例|備考|
|:--:|:------:|:----------------:|:----------------:|:--:|
|`/healthz`|`GET`|||動作・疎通確認用|
|`/operation`|`GET`/`PUT`|`application/json`|`{"operation": "f"}`|operation はマイコン実装で使われている s, f, b, r, l のいずれか|

