#!/usr/bin/env bash
# 校验 dbx/hubx/mysqlx（MySQL） 是否真的可用（协议级探活，不只是端口通）
set -uo pipefail
cd "$(dirname "$0")/.."

DOCKER="$(command -v docker || true)"
if [ -z "$DOCKER" ] && [ -x /usr/local/bin/docker ]; then
  DOCKER=/usr/local/bin/docker
fi
if [ -z "$DOCKER" ]; then
  echo "ERROR: 未找到 docker CLI。"
  exit 1
fi

HOST_PORT="${HOST_PORT:-3306}"
fail=0

check() {
  local desc="$1"; shift
  if out="$("$@" 2>&1)"; then
    echo "  OK   ${desc}"
    [ -n "${VERBOSE:-}" ] && [ -n "$out" ] && echo "       $out"
  else
    echo "  FAIL ${desc}"
    [ -n "$out" ] && echo "       $out"
    fail=1
  fi
}

echo "Verifying dbx/hubx/mysqlx（MySQL）..."
check "mysqladmin ping" $DOCKER exec hubx-dbx-mysql mysqladmin ping -h localhost -uroot -prootpass
check "testdb 可查询" $DOCKER exec hubx-dbx-mysql mysql -uroot -prootpass -e "SELECT 1" testdb
if [ "$fail" -eq 0 ]; then
  echo "ALL CHECKS PASSED"
else
  echo "SOME CHECKS FAILED（先看日志: $DOCKER compose logs）"
  exit 1
fi
