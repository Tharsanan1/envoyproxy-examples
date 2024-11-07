docker-compose -f docker-compose-go.yaml run --rm go_plugin_compile
docker-compose pull
export DOCKER_BUILDKIT=1 
docker-compose up --build -d