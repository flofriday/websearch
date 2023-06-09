# websearch

![Screenshot](screenshot.png)

Let's build a search engine for the web, just for fun. 🥳

## Build it yourself

```
go build
./websearch index
./websearch search "Linux"
```

## Build with docker

```
# Build the image
docker build -t "websearch" .

# Build the index
docker volume create websearch_index
docker run \
    --rm \
    --mount source=websearch_index,target=/app/data \
    websearch index --sqlite data/index.db

# Serve the index 
docker run \
    --rm \
    -p 8080:8080 \
    --mount source=websearch_index,target=/app/data \
    websearch server --sqlite data/index.db
```

## Architecture
![Architecture](architecture.png)
