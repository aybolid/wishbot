docker load -i wishbot.tar.gz
docker run -d \
    --env-file .env \
    --volume $(pwd)/logs:/app/logs \
    --volume $(pwd)/data:/app/data \
    --name wishbot \
    wishbot