docker compose build
docker save -o wishbot.tar wishbot
gzip wishbot.tar