# Docker

## 基本操作
```bash
# コンテナ一覧
docker ps
docker ps -a

# イメージ一覧
docker images

# コンテナ起動
docker run -it <image>
docker run -d -p 8080:80 <image>

# コンテナ停止・削除
docker stop <container>
docker rm <container>
```

## ビルド
```bash
# イメージビルド
docker build -t <name> .
docker build -t <name>:<tag> .

# キャッシュなしでビルド
docker build --no-cache -t <name> .
```

## ログ・デバッグ
```bash
# ログ確認
docker logs <container>
docker logs -f <container>

# コンテナに入る
docker exec -it <container> /bin/bash
docker exec -it <container> sh
```

## Docker Compose
```bash
# 起動
docker-compose up
docker-compose up -d

# 停止
docker-compose down

# 再ビルド
docker-compose up --build

# ログ
docker-compose logs -f
```

## クリーンアップ
```bash
# 停止中のコンテナ削除
docker container prune

# 未使用イメージ削除
docker image prune

# 全部クリーン
docker system prune -a
```
## ボリューム
```bash
# ボリューム一覧
docker volume ls

# ボリューム作成
docker volume create <name>

# ボリュームマウントして起動
docker run -v <volume>:/data <image>

# ホストディレクトリをマウント
docker run -v $(pwd):/app <image>
```

## ネットワーク
```bash
# ネットワーク一覧
docker network ls

# ネットワーク作成
docker network create <name>

# ネットワーク指定して起動
docker run --network <name> <image>
```

## レジストリ
```bash
# ログイン
docker login

# プッシュ
docker push <image>:<tag>

# プル
docker pull <image>:<tag>
```

## よくあるエラー
- `port is already allocated` → 別のポート使うか、既存コンテナ停止
- `no space left on device` → `docker system prune -a` でクリーンアップ
- `permission denied` → sudo つけるか、docker グループに追加