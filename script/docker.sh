#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
compose=(docker compose -f docker/lx-2/docker-compose.yml)

case "${1:-help}" in
     build)
          "${compose[@]}" build
     ;;
     up)
          "${compose[@]}" up --build -d
          "${compose[@]}" ps
     ;;
     down)
          "${compose[@]}" down
     ;;
     restart)
          "${compose[@]}" restart
     ;;
     logs)
          "${compose[@]}" logs -f leoxy
     ;;
     status)
          "${compose[@]}" ps
     ;;
     health)
          "${compose[@]}" exec leoxy wget -q -O - http://127.0.0.1:4444/health/live
     ;;
     clean)
          "${compose[@]}" down --rmi local
     ;;
     *)
          echo "Usage: $0 {build|up|down|restart|logs|status|health|clean}"
     ;;
esac
