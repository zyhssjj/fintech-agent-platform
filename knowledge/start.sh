#!/usr/bin/env bash
# 一键启动科技金融企业知识库（Elasticsearch + 应用服务）

set -Eeuo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

REBUILD=false

case "${1:-}" in
  "")
    ;;
  --build|--rebuild)
    REBUILD=true
    ;;
  -h|--help)
    printf '用法：%s [--build]\n' "$(basename "$0")"
    printf '  不带参数：快速启动已有镜像，不主动重新构建。\n'
    printf '  --build：  更新代码或 Dockerfile 后重新构建并启动。\n'
    exit 0
    ;;
  *)
    printf '未知参数：%s\n用法：%s [--build]\n' "$1" "$(basename "$0")" >&2
    exit 2
    ;;
esac

fail() {
  printf '\n启动失败：%s\n' "$1" >&2
  exit 1
}

if ! command -v docker >/dev/null 2>&1; then
  fail "未检测到 Docker。请先安装并启动 Docker Desktop。"
fi

if ! docker info >/dev/null 2>&1; then
  fail "Docker 服务尚未运行。请打开 Docker Desktop 后重试。"
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE=(docker-compose)
else
  fail "未检测到 Docker Compose。请升级 Docker Desktop 或安装 docker-compose。"
fi

if [[ ! -f .env ]]; then
  printf '提示：未找到 .env，将以 docker-compose.yml 中的默认配置启动。\n'
  printf '      如需使用云端模型，请先从 .env.example 创建 .env 并填写模型配置。\n'
fi

if [[ "$REBUILD" == true ]]; then
  printf '正在重新构建并启动科技金融企业知识库…\n'
  "${COMPOSE[@]}" up -d --build
else
  printf '正在快速启动科技金融企业知识库…\n'
  "${COMPOSE[@]}" up -d
fi

printf '正在等待应用服务就绪…\n'
for _ in $(seq 1 60); do
  if curl -fsS http://127.0.0.1:8201/health >/dev/null 2>&1; then
    printf '\n启动成功。\n'
    printf '知识库地址：http://127.0.0.1:8201\n'
    printf '健康检查：  http://127.0.0.1:8201/health\n'
    printf 'Elasticsearch：http://127.0.0.1:9202\n\n'
    "${COMPOSE[@]}" ps
    exit 0
  fi
  sleep 2
done

printf '\n应用在 120 秒内未就绪，当前服务状态：\n' >&2
"${COMPOSE[@]}" ps >&2
printf '\n可使用以下命令查看日志：\n  %s logs --tail=100 app\n' "${COMPOSE[*]}" >&2
exit 1
