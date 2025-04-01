# Podconfig

Podconfig is a lightweight web configuration interface for [Podsync](https://github.com/Podsync/podsync). It allows you to easily add, remove, and manage feeds for your Podsync server through a simple web interface.

## Features

- **Feed Management:** Add and remove YouTube channels (feeds) via the web interface.
- **Configuration Editing:** Automatically updates Podsync’s TOML configuration file.
- **Docker Integration:** Reloads the Podsync Docker container after changes.

## Prerequisites

- [Go](https://golang.org/dl/) (version 1.16 or higher recommended)
- Podsync installed and configured via Docker

## Installation

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/Takenobou/podconfig.git
   cd podconfig
   ```

2. **Build the Application:**

   Build the executable using Go:

   ```bash
   go build -o podconfig ./cmd/podconfig
   ```

3. **Configuration:**

   Podconfig reads its settings from environment variables. You can set these in your deployment environment or in your systemd service file.

   - `PODSYNC_CONFIG_PATH`: Path to your Podsync configuration file (default: `../config.toml`).
   - `DOCKER_CONTAINER_NAME`: Name of your Podsync Docker container (default: `podsync`).
   - `SERVER_PORT`: Port on which the web server will run (default: `8080`).

## Running the Application

To run the application manually, set the required environment variables and start the server:

```bash
export PODSYNC_CONFIG_PATH=/path/to/config.toml
export DOCKER_CONTAINER_NAME=podsync
export SERVER_PORT=8080
./podconfig
```

Then, open your web browser and navigate to `http://localhost:8080`.

## Deployment with systemd

To run Podconfig as a systemd service, create a service file (e.g., `/etc/systemd/system/podconfig.service`) with the following content:

```ini
[Unit]
Description=Podconfig Web Server
After=network.target

[Service]
User=user
WorkingDirectory=/path/to/podconfig
Environment=PODSYNC_CONFIG_PATH=/path/to/podsync/config.toml
Environment=DOCKER_CONTAINER_NAME=podsync
Environment=SERVER_PORT=8080
ExecStart=/path/to/podconfig/podconfig
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### To Enable and Start the Service:

1. **Reload systemd to pick up the new service file:**

   ```bash
   sudo systemctl daemon-reload
   ```

2. **Enable the service to start at boot:**

   ```bash
   sudo systemctl enable podconfig.service
   ```

3. **Start the service:**

   ```bash
   sudo systemctl start podconfig.service
   ```

4. **Check the status:**

   ```bash
   sudo systemctl status podconfig.service
   ```

## Deployment with Docker

You can also deploy Podconfig and Podsync using Docker Compose. The `docker-compose.yml` file is included in the repository and defines both services.

### Example `docker-compose.yml`:

```yaml
services:
  podsync:
    container_name: podsync
    image: ghcr.io/mxpv/podsync:latest
    restart: always
    ports:
      - 8050:8050
    volumes:
      - ${CONFIG_PATH}/podsync:/app/data/
      - ${CONFIG_PATH}/podsync/config.toml:/app/config.toml

  podconfig:
    container_name: podconfig
    image: ghcr.io/takenobou/podconfig:latest
    restart: unless-stopped
    depends_on:
      - podsync
    ports:
      - "8080:8080"
    environment:
      PODSYNC_CONFIG_PATH: "/config/config.toml"
      DOCKER_CONTAINER_NAME: "podsync"
      SERVER_PORT: "8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ${CONFIG_PATH}/podsync/config.toml:/config/config.toml
```

### Running the services

1. Set the `CONFIG_PATH` environment variable to point to your desired configuration directory:
   ```bash
   export CONFIG_PATH=/path/to/your/config
   ```

2. Start the services using Docker Compose:
   ```bash
   docker-compose up -d
   ```

3. Visit `http://localhost:8080` to access Podconfig.
