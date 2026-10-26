#!/usr/bin/env bash
set -euo pipefail

url=127.0.0.1:4444

case "${1:-help}" in
     build)
          docker compose build
     ;;
     up)
          docker compose up build -d
          docker compose ps
     ;;
     down)
          docker compose down
     ;;
     restart)
          docker compose restart
     ;;
     logs)
          docker compose logs -f leoxy
     ;;
     status)
          docker compose ps
     ;;   
     health)
          docker compose exec leoxy wget -q -O - \ http://${url}/health/live
     ;;
     clean)
          docker compose down --rmi local
     ;;
     help|*)
          echo "Usage: ./docker.sh for {
               build ||
               up ||
               down ||
               restart ||
               logs ||
               status ||
               health ||
               clean
          }"

echo "Docker Done!! - 🦐"
