# サーバー実装について

このディレクトリは、HTTP リクエストを受け取り UART でマイコンと通信したうえでレスポンスを返す実装が入っています。


# 使用方法

1. submodule をすべて clone する

    - router-rc-tank を clone するときに `git clone --recursive https://github.com/puhitaku/router-rc-tank.git` とすると submodule がすべて落ちてくる
    - もし既に router-rc-tank を clone していた場合（serial や nanoweb の中になにもない場合）は `git submodule update --init --recursive` ですべて落ちてくる

2. 実機に転送する

    ```sh
    $ scp -r api root@{address_of_the_router}:~
    ```

3. 実機で実行する

    ```sh
    $ ssh root@{address_of_the_router}
    $ micropython api/main.py
    ```

4. リクエストを送る

    ```sh
    $ curl {address_of_the_router}:8080/healthz
    {"message": "I'm as ready as I'll ever be!"}

    $ curl -X PUT -H "Content-Type: application/json" -d '{"operation": "f"}' {address_of_the_router}:8080/operation
    {"operation": "f", "error": null}
    ```

# 仕様

ポート番号: 8080

|パス|メソッド|Content-Type (PUT)|リクエストボディ例|備考|
|:--:|:------:|:----------------:|:----------------:|:--:|
|`/healthz`|`GET`|||動作・疎通確認用|
|`/operation`|`GET`/`PUT`|`application/json`|`{"operation": "f"}`|operation はマイコン実装で使われている s, f, b, r, l のいずれか|

