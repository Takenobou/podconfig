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
   git clone https://github.com/yourusername/podconfig.git
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
User=root
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

